package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog/log"

	"github.com/jackc/pgx/v5/pgxpool"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waTypes "go.mau.fi/whatsmeow/types"
	waEvents "go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/example/wpp-wave-bot/internal/contacts"
	"github.com/example/wpp-wave-bot/internal/groups"
	"github.com/example/wpp-wave-bot/internal/messages"
	"github.com/example/wpp-wave-bot/internal/rabbitmq"
)

// OutgoingMessage represents a message consumed from the queue to be sent.
type OutgoingMessage struct {
	CompanyID string `json:"company_id"`
	Type      string `json:"type"`
	To        string `json:"to"`
	Message   string `json:"message"`
	MediaURL  string `json:"media_url"`
	Filename  string `json:"filename"`
}

// IncomingMessage represents a received WhatsApp message published to the queue.
type IncomingMessage struct {
	CompanyID string    `json:"company_id"`
	From      string    `json:"from"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Service manages WhatsApp sessions and message flow.
type Service struct {
	db      *pgxpool.Pool
	mq      *rabbitmq.RabbitMQ
	store   *sqlstore.Container
	clients map[string]*whatsmeow.Client

	msgRepo     *messages.Repository
	contactRepo *contacts.Repository
	groupRepo   *groups.Repository
}

// Sessions returns the list of company IDs with active clients.
func (s *Service) Sessions() []string {
	ids := make([]string, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	return ids
}

// Logout disconnects the client's session and removes it from storage.
func (s *Service) Logout(ctx context.Context, companyID string) error {
	cli, ok := s.clients[companyID]
	if !ok {
		return fmt.Errorf("session not found")
	}
	if err := cli.Logout(ctx); err != nil {
		return err
	}
	delete(s.clients, companyID)
	_, err := s.db.Exec(ctx, "DELETE FROM sessions WHERE company_id=$1", companyID)
	return err
}

// Connect ensures a client for the company is connected. When authentication is
// required, the first QR code string is returned so it can be rendered to the
// user.
func (s *Service) Connect(ctx context.Context, companyID string) (string, error) {
	_, qr, err := s.getClient(ctx, companyID)
	return qr, err
}

// Send dispatches a message immediately using WhatsApp.
func (s *Service) Send(ctx context.Context, msg *OutgoingMessage) error {
	cli, _, err := s.getClient(ctx, msg.CompanyID)
	if err != nil {
		return err
	}
	return s.sendMessage(ctx, cli, msg)
}

// New creates a new Service instance using the given Postgres URL for the whatsmeow store.
func New(db *pgxpool.Pool, dbURL string, mq *rabbitmq.RabbitMQ) (*Service, error) {
	container, err := sqlstore.New(context.Background(), "pgx", dbURL, waLog.Noop)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:          db,
		mq:          mq,
		store:       container,
		clients:     make(map[string]*whatsmeow.Client),
		msgRepo:     messages.NewRepository(db),
		contactRepo: contacts.NewRepository(db),
		groupRepo:   groups.NewRepository(db),
	}, nil
}

// Start begins consuming messages from RabbitMQ.
func (s *Service) Start(ctx context.Context) error {
	msgs, err := s.mq.Consume("wpp:send")
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d := <-msgs:
			var m OutgoingMessage
			if err := json.Unmarshal(d.Body, &m); err != nil {
				log.Error().Err(err).Msg("invalid message payload")
				continue
			}
			cli, _, err := s.getClient(ctx, m.CompanyID)
			if err != nil {
				log.Error().Err(err).Msg("failed to get client")
				continue
			}
			if err := s.sendMessage(ctx, cli, &m); err != nil {
				log.Error().Err(err).Msg("failed to send message")
			}
		}
	}
}

// getClient returns or creates a WhatsApp client for the given company.
// If a new login is required the first QR code string is returned.
func (s *Service) getClient(ctx context.Context, companyID string) (*whatsmeow.Client, string, error) {
	if c, ok := s.clients[companyID]; ok {
		return c, "", nil
	}

	var jidStr string
	_ = s.db.QueryRow(ctx, "SELECT data FROM sessions WHERE company_id=$1", companyID).Scan(&jidStr)

	var device *store.Device
	if jidStr != "" {
		jid, err := waTypes.ParseJID(jidStr)
		if err == nil {
			device, _ = s.store.GetDevice(ctx, jid)
		}
	}
	if device == nil {
		device = s.store.NewDevice()
	}

	cli := whatsmeow.NewClient(device, waLog.Stdout("client"+companyID, "INFO", true))
	cli.AddEventHandler(s.eventHandler(companyID, cli))

	var qr string
	if cli.Store.ID == nil {
		qrChan, _ := cli.GetQRChannel(ctx)
		if err := cli.Connect(); err != nil {
			return nil, "", err
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				log.Info().Str("company_id", companyID).Msgf("scan QR: %s", evt.Code)
				s.publishSessionEvent(companyID, "qr", evt.Code)
				if qr == "" {
					qr = evt.Code
				}
			} else {
				log.Info().Str("company_id", companyID).Msgf("login event: %s", evt.Event)
			}
		}
		if cli.Store.ID != nil {
			_, err := s.db.Exec(ctx, `
                INSERT INTO sessions (company_id, data)
                VALUES ($1, $2)
                ON CONFLICT (company_id) DO UPDATE SET data=EXCLUDED.data
            `, companyID, []byte(cli.Store.ID.String()))
			if err != nil {
				log.Error().Err(err).Msg("failed to store session jid")
			}
		}
	} else if err := cli.Connect(); err != nil {
		return nil, "", err
	}

	s.clients[companyID] = cli
	return cli, qr, nil
}

func (s *Service) eventHandler(companyID string, cli *whatsmeow.Client) func(evt any) {
	return func(evt any) {
		switch v := evt.(type) {
		case *waEvents.Message:
			s.handleIncoming(companyID, cli, v)
		case *waEvents.Disconnected:
			log.Warn().Str("company_id", companyID).Msg("client disconnected")
			s.publishSessionEvent(companyID, "disconnected", "")
		case *waEvents.Connected:
			log.Info().Str("company_id", companyID).Msg("client connected")
			s.publishSessionEvent(companyID, "connected", "")
		}
	}
}

func (s *Service) handleIncoming(companyID string, cli *whatsmeow.Client, evt *waEvents.Message) {
	msg := evt.Message
	var msgType, content string
	switch {
	case msg.GetConversation() != "":
		msgType = "text"
		content = msg.GetConversation()
	case msg.GetImageMessage() != nil:
		msgType = "image"
		content = msg.GetImageMessage().GetCaption()
	case msg.GetAudioMessage() != nil:
		msgType = "audio"
	case msg.GetDocumentMessage() != nil:
		msgType = "document"
		content = msg.GetDocumentMessage().GetFileName()
	default:
		msgType = "other"
	}

	_ = s.msgRepo.Save(context.Background(), companyID, string(evt.Info.ID), evt.Info.Sender.String(), evt.Info.Chat.String(), msgType, content)
	_ = s.contactRepo.Upsert(context.Background(), companyID, evt.Info.Sender.String(), evt.Info.PushName, "")
	if evt.Info.Chat.Server == waTypes.GroupServer {
		_ = s.groupRepo.Upsert(context.Background(), companyID, evt.Info.Chat.String(), evt.Info.PushName)
	}

	out := IncomingMessage{
		CompanyID: companyID,
		From:      evt.Info.Sender.String(),
		Type:      msgType,
		Message:   content,
		Timestamp: evt.Info.Timestamp.UTC(),
	}
	body, _ := json.Marshal(out)
	_ = s.mq.Publish("", "wpp:received", body)
}

func (s *Service) sendMessage(ctx context.Context, cli *whatsmeow.Client, m *OutgoingMessage) error {
	to, err := waTypes.ParseJID(m.To)
	if err != nil {
		return err
	}

	var msg *waProto.Message

	switch m.Type {
	case "text":
		msg = &waProto.Message{Conversation: proto.String(m.Message)}
	case "image":
		data, err := download(m.MediaURL)
		if err != nil {
			return err
		}
		up, err := cli.Upload(ctx, data, whatsmeow.MediaImage)
		if err != nil {
			return err
		}
		msg = &waProto.Message{ImageMessage: &waProto.ImageMessage{
			Caption:       proto.String(m.Message),
			Mimetype:      proto.String(http.DetectContentType(data)),
			URL:           &up.URL,
			DirectPath:    &up.DirectPath,
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    &up.FileLength,
		}}
	case "audio":
		data, err := download(m.MediaURL)
		if err != nil {
			return err
		}
		up, err := cli.Upload(ctx, data, whatsmeow.MediaAudio)
		if err != nil {
			return err
		}
		msg = &waProto.Message{AudioMessage: &waProto.AudioMessage{
			Mimetype:      proto.String(http.DetectContentType(data)),
			URL:           &up.URL,
			DirectPath:    &up.DirectPath,
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    &up.FileLength,
		}}
	case "document":
		data, err := download(m.MediaURL)
		if err != nil {
			return err
		}
		up, err := cli.Upload(ctx, data, whatsmeow.MediaDocument)
		if err != nil {
			return err
		}
		msg = &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
			FileName:      proto.String(m.Filename),
			Mimetype:      proto.String(http.DetectContentType(data)),
			URL:           &up.URL,
			DirectPath:    &up.DirectPath,
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    &up.FileLength,
		}}
	default:
		return fmt.Errorf("unknown message type %s", m.Type)
	}

	resp, err := cli.SendMessage(ctx, to, msg)
	if err == nil {
		_ = s.msgRepo.Save(ctx, m.CompanyID, resp.ID, cli.Store.ID.String(), to.String(), m.Type, m.Message)
	}
	return err
}

func (s *Service) publishSessionEvent(companyID, status, code string) {
	evt := map[string]string{
		"company_id": companyID,
		"status":     status,
	}
	if code != "" {
		evt["code"] = code
	}
	body, _ := json.Marshal(evt)
	if err := s.mq.Publish("", "wpp:sessions", body); err != nil {
		log.Error().Err(err).Msg("failed to publish session event")
	}
}

func download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
