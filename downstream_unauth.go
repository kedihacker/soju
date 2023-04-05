package soju

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/irc.v4"

	"git.sr.ht/~emersion/soju/auth"
	"git.sr.ht/~emersion/soju/xirc"
)

/*func serverSASLMechanisms(srv *Server) []string {
	var l []string
	if _, ok := srv.Config().Auth.(auth.PlainAuthenticator); ok {
		l = append(l, "PLAIN")
	}
	if _, ok := srv.Config().Auth.(auth.OAuthBearerAuthenticator); ok {
		l = append(l, "OAUTHBEARER")
	}
	return l
}*/

// unauthDownstreamConn is a downstream connection prior to RPL_WELCOME.
type unauthDownstreamConn struct {
	conn

	id uint64

	registered bool
	nick       string
	username   string
	pass       string

	networkName string
	networkID   int64
	clientName  string

	negotiatingCaps bool
	capVersion      int
	caps            xirc.CapRegistry
	sasl            *downstreamSASL // nil unless SASL is underway
}

func newUnauthDownstreamConn(srv *Server, ic ircConn, id uint64) *downstreamConn {
	remoteAddr := ic.RemoteAddr().String()
	logger := &prefixLogger{srv.Logger, fmt.Sprintf("downstream %q: ", remoteAddr)}
	udc := &downstreamConn{
		conn: *newConn(srv, ic, &connOptions{Logger: logger}),
		id:   id,
		caps: xirc.NewCapRegistry(),
	}
	for k, v := range permanentDownstreamCaps {
		udc.caps.Available[k] = v
	}
	udc.caps.Available["sasl"] = strings.Join(serverSASLMechanisms(udc.srv), ",")
	// TODO: this is racy, we should only enable chathistory after
	// authentication and then check that user.msgStore implements
	// chatHistoryMessageStore
	switch srv.Config().LogDriver {
	case "fs", "db":
		udc.caps.Available["draft/chathistory"] = ""
		udc.caps.Available["soju.im/search"] = ""
	}
	return udc
}

func (udc *unauthDownstreamConn) runUntilRegistered() error {
	ctx, cancel := context.WithTimeout(context.TODO(), downstreamRegisterTimeout)
	defer cancel()

	// Close the connection with an error if the deadline is exceeded
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err == context.DeadlineExceeded {
			udc.SendMessage(context.TODO(), &irc.Message{
				Command: "ERROR",
				Params:  []string{"Connection registration timed out"},
			})
			udc.Close()
		}
	}()

	for !udc.registered {
		msg, err := udc.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read IRC command: %w", err)
		}

		err = udc.handleMessage(ctx, msg)
		if ircErr, ok := err.(ircError); ok {
			ircErr.Message.Prefix = udc.srv.prefix()
			udc.SendMessage(ctx, ircErr.Message)
		} else if err != nil {
			return fmt.Errorf("failed to handle IRC command %q: %v", msg, err)
		}
	}

	return nil
}

func (udc *unauthDownstreamConn) handleMessage(ctx context.Context, msg *irc.Message) error {
	switch msg.Command {
	case "NICK":
		if err := parseMessageParams(msg, &udc.nick); err != nil {
			return err
		}
	case "USER":
		if err := parseMessageParams(msg, &udc.username, nil, nil, nil); err != nil {
			return err
		}
	case "PASS":
		if err := parseMessageParams(msg, &udc.pass); err != nil {
			return err
		}
	case "CAP":
		return udc.handleCap(msg)
	case "AUTHENTICATE":
		// TODO
	case "BOUNCER":
		var subcommand string
		if err := parseMessageParams(msg, &subcommand); err != nil {
			return err
		}
		if strings.ToUpper(subcommand) != "BIND" {
			return ircError{&irc.Message{
				Command: "FAIL",
				Params:  []string{"BOUNCER", "UNKNOWN_COMMAND", subcommand, "Unknown subcommand"},
			}}
		}

		var idStr string
		if err := parseMessageParams(msg, nil, &idStr); err != nil {
			return err
		}

		id, err := parseBouncerNetID(subcommand, idStr)
		if err != nil {
			return err
		}
		udc.networkID = id
	default:
		return newUnknownCommandError(msg.Command)
	}
	if udc.nick != "" && udc.username != "" && !udc.negotiatingCaps {
		return udc.register(ctx)
	}
	return nil
}

func (udc *unauthDownstreamConn) handleCap(msg *irc.Message) error {
	var cmd string
	if err := parseMessageParams(msg, &cmd); err != nil {
		return err
	}
	args := msg.Params[1:]

	switch cmd = strings.ToUpper(cmd); cmd {
	case "LS":
	case "END":
		dc.negotiatingCaps = false
	default:
		return ircError{&irc.Message{
			Command: xirc.ERR_INVALIDCAPCMD,
			Params:  []string{"*", cmd, "Unknown CAP command"},
		}}
	}
}

func (udc *unauthDownstreamConn) register(ctx context.Context) error {
	return nil
}
