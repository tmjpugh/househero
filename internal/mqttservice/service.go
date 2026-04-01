package mqttservice

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Topic constants used for publishing events and subscribing to commands.
const (
	TopicTicketCreated   = "househero/tickets/created"
	TopicTicketUpdated   = "househero/tickets/updated"
	TopicCommentAdded    = "househero/tickets/comment_added"
	TopicInventoryCreated = "househero/inventory/created"
	TopicInventoryUpdated = "househero/inventory/updated"

	// Command topics (incoming)
	TopicCmdTicketCreate = "househero/commands/tickets/create"
	TopicCmdTicketDetail = "househero/commands/tickets/detail"

	// Response topic prefix — callers append their request_id.
	TopicResponsePrefix = "househero/responses/"
)

// Service wraps an MQTT client and exposes publish/subscribe helpers.
type Service struct {
	client  mqtt.Client
	handler CommandHandler
}

// CommandHandler receives parsed MQTT commands and must be implemented by the caller.
type CommandHandler interface {
	// HandleCreateTicket is called when a create-ticket command arrives.
	// payload contains the raw JSON from the MQTT message.
	HandleCreateTicket(payload []byte) (response interface{}, err error)

	// HandleTicketDetail is called when a ticket-detail command arrives.
	// payload contains the raw JSON from the MQTT message.
	HandleTicketDetail(payload []byte) (response interface{}, err error)
}

// New creates and connects a new MQTT Service.
// broker should be in the form "tcp://host:port".
// If the broker address is empty the service is disabled and all publish calls
// are no-ops so the application runs fine without MQTT configured.
func New(broker, clientID, username, password string, handler CommandHandler) (*Service, error) {
	if broker == "" {
		log.Println("MQTT: no broker configured, MQTT disabled")
		return &Service{}, nil
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(60 * time.Second).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Printf("MQTT: connection lost: %v", err)
		}).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Println("MQTT: connected")
		})

	if username != "" {
		opts.SetUsername(username).SetPassword(password)
	}

	client := mqtt.NewClient(opts)
	svc := &Service{client: client, handler: handler}

	token := client.Connect()
	if token.WaitTimeout(10*time.Second) && token.Error() != nil {
		return nil, fmt.Errorf("MQTT connect: %w", token.Error())
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("MQTT: failed to connect to broker %s", broker)
	}

	svc.subscribeCommands()
	return svc, nil
}

// IsEnabled returns true when an MQTT broker is connected.
func (s *Service) IsEnabled() bool {
	return s.client != nil && s.client.IsConnected()
}

// Publish sends a JSON-encoded payload to the given topic.
// The call is a no-op when MQTT is disabled.
func (s *Service) Publish(topic string, payload interface{}) {
	if !s.IsEnabled() {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("MQTT publish marshal error on %s: %v", topic, err)
		return
	}
	token := s.client.Publish(topic, 1, false, data)
	token.WaitTimeout(5 * time.Second)
	if token.Error() != nil {
		log.Printf("MQTT publish error on %s: %v", topic, token.Error())
	}
}

// subscribeCommands registers listeners for the incoming command topics.
func (s *Service) subscribeCommands() {
	if s.handler == nil {
		return
	}
	s.subscribe(TopicCmdTicketCreate, s.onCreateTicket)
	s.subscribe(TopicCmdTicketDetail, s.onTicketDetail)
}

func (s *Service) subscribe(topic string, fn mqtt.MessageHandler) {
	token := s.client.Subscribe(topic, 1, fn)
	token.WaitTimeout(5 * time.Second)
	if token.Error() != nil {
		log.Printf("MQTT subscribe error on %s: %v", topic, token.Error())
	} else {
		log.Printf("MQTT: subscribed to %s", topic)
	}
}

// onCreateTicket handles househero/commands/tickets/create messages.
// Expected payload:
//
//	{
//	  "request_id": "optional-string",   // response will be published to househero/responses/<request_id>
//	  "home_id":    1,                   // required
//	  "title":      "Leaky faucet",      // required
//	  "type":       "maintenance",       // optional, default "maintenance"
//	  "priority":   "medium",            // optional, default "medium"
//	  "requester":  "Alice",             // optional
//	  "room":       "Bathroom",          // optional
//	  "description":"...",               // optional
//	  "estimated_cost": "150.00"         // optional
//	}
func (s *Service) onCreateTicket(_ mqtt.Client, msg mqtt.Message) {
	resp, err := s.handler.HandleCreateTicket(msg.Payload())
	s.publishCommandResponse(msg.Payload(), resp, err)
}

// onTicketDetail handles househero/commands/tickets/detail messages.
// Expected payload:
//
//	{
//	  "request_id":    "optional-string",
//	  "ticket_number": 42,               // required (ticket number within a home)
//	  "home_id":       1                 // required
//	}
func (s *Service) onTicketDetail(_ mqtt.Client, msg mqtt.Message) {
	resp, err := s.handler.HandleTicketDetail(msg.Payload())
	s.publishCommandResponse(msg.Payload(), resp, err)
}

// publishCommandResponse sends the handler result back to the response topic.
// The request_id field in the raw payload is used to build the topic.
func (s *Service) publishCommandResponse(rawPayload []byte, resp interface{}, handlerErr error) {
	var base struct {
		RequestID string `json:"request_id"`
	}
	// Best-effort parse to extract request_id; errors leave RequestID as empty string.
	if err := json.Unmarshal(rawPayload, &base); err != nil {
		log.Printf("MQTT: could not parse request_id from payload: %v", err)
	}

	topic := TopicResponsePrefix + base.RequestID
	if base.RequestID == "" {
		topic = TopicResponsePrefix + "default"
	}

	if handlerErr != nil {
		s.Publish(topic, map[string]string{"error": handlerErr.Error()})
		return
	}
	s.Publish(topic, resp)
}

// Close disconnects the MQTT client cleanly.
func (s *Service) Close() {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(500)
	}
}
