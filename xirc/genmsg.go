package xirc

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/irc.v3"
)

func GenerateJoin(channels, keys []string) []*irc.Message {
	// Put channels with a key first
	js := joinSorter{channels, keys}
	sort.Sort(&js)

	// Two spaces because there are three words (JOIN, channels and keys)
	maxLength := maxMessageLength - (len("JOIN") + 2)

	var msgs []*irc.Message
	var channelsBuf, keysBuf strings.Builder
	for i, channel := range channels {
		key := keys[i]

		n := channelsBuf.Len() + keysBuf.Len() + 1 + len(channel)
		if key != "" {
			n += 1 + len(key)
		}

		if channelsBuf.Len() > 0 && n > maxLength {
			// No room for the new channel in this message
			params := []string{channelsBuf.String()}
			if keysBuf.Len() > 0 {
				params = append(params, keysBuf.String())
			}
			msgs = append(msgs, &irc.Message{Command: "JOIN", Params: params})
			channelsBuf.Reset()
			keysBuf.Reset()
		}

		if channelsBuf.Len() > 0 {
			channelsBuf.WriteByte(',')
		}
		channelsBuf.WriteString(channel)
		if key != "" {
			if keysBuf.Len() > 0 {
				keysBuf.WriteByte(',')
			}
			keysBuf.WriteString(key)
		}
	}
	if channelsBuf.Len() > 0 {
		params := []string{channelsBuf.String()}
		if keysBuf.Len() > 0 {
			params = append(params, keysBuf.String())
		}
		msgs = append(msgs, &irc.Message{Command: "JOIN", Params: params})
	}

	return msgs
}

type joinSorter struct {
	channels []string
	keys     []string
}

func (js *joinSorter) Len() int {
	return len(js.channels)
}

func (js *joinSorter) Less(i, j int) bool {
	if (js.keys[i] != "") != (js.keys[j] != "") {
		// Only one of the channels has a key
		return js.keys[i] != ""
	}
	return js.channels[i] < js.channels[j]
}

func (js *joinSorter) Swap(i, j int) {
	js.channels[i], js.channels[j] = js.channels[j], js.channels[i]
	js.keys[i], js.keys[j] = js.keys[j], js.keys[i]
}

func GenerateIsupport(prefix *irc.Prefix, nick string, tokens []string) []*irc.Message {
	maxTokens := maxMessageParams - 2 // 2 reserved params: nick + text

	var msgs []*irc.Message
	for len(tokens) > 0 {
		var msgTokens []string
		if len(tokens) > maxTokens {
			msgTokens = tokens[:maxTokens]
			tokens = tokens[maxTokens:]
		} else {
			msgTokens = tokens
			tokens = nil
		}

		msgs = append(msgs, &irc.Message{
			Prefix:  prefix,
			Command: irc.RPL_ISUPPORT,
			Params:  append(append([]string{nick}, msgTokens...), "are supported"),
		})
	}

	return msgs
}

func GenerateMOTD(prefix *irc.Prefix, nick string, motd string) []*irc.Message {
	var msgs []*irc.Message
	msgs = append(msgs, &irc.Message{
		Prefix:  prefix,
		Command: irc.RPL_MOTDSTART,
		Params:  []string{nick, fmt.Sprintf("- Message of the Day -")},
	})

	for _, l := range strings.Split(motd, "\n") {
		msgs = append(msgs, &irc.Message{
			Prefix:  prefix,
			Command: irc.RPL_MOTD,
			Params:  []string{nick, l},
		})
	}

	msgs = append(msgs, &irc.Message{
		Prefix:  prefix,
		Command: irc.RPL_ENDOFMOTD,
		Params:  []string{nick, "End of /MOTD command."},
	})

	return msgs
}

func GenerateMonitor(subcmd string, targets []string) []*irc.Message {
	maxLength := maxMessageLength - len("MONITOR "+subcmd+" ")

	var msgs []*irc.Message
	var buf []string
	n := 0
	for _, target := range targets {
		if n+len(target)+1 > maxLength {
			msgs = append(msgs, &irc.Message{
				Command: "MONITOR",
				Params:  []string{subcmd, strings.Join(buf, ",")},
			})
			buf = buf[:0]
			n = 0
		}

		buf = append(buf, target)
		n += len(target) + 1
	}

	if len(buf) > 0 {
		msgs = append(msgs, &irc.Message{
			Command: "MONITOR",
			Params:  []string{subcmd, strings.Join(buf, ",")},
		})
	}

	return msgs
}

func GenerateNamesReply(prefix *irc.Prefix, nick string, channel string, status ChannelStatus, members []string) []*irc.Message {
	emptyNameReply := irc.Message{
		Prefix:  prefix,
		Command: irc.RPL_NAMREPLY,
		Params:  []string{nick, string(status), channel, ""},
	}
	maxLength := maxMessageLength - len(emptyNameReply.String())

	var msgs []*irc.Message
	var buf strings.Builder
	for _, s := range members {
		n := buf.Len() + 1 + len(s)
		if buf.Len() != 0 && n > maxLength {
			// There's not enough space for the next space + nick
			msgs = append(msgs, &irc.Message{
				Prefix:  prefix,
				Command: irc.RPL_NAMREPLY,
				Params:  []string{nick, string(status), channel, buf.String()},
			})
			buf.Reset()
		}

		if buf.Len() != 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(s)
	}

	if buf.Len() != 0 {
		msgs = append(msgs, &irc.Message{
			Prefix:  prefix,
			Command: irc.RPL_NAMREPLY,
			Params:  []string{nick, string(status), channel, buf.String()},
		})
	}

	msgs = append(msgs, &irc.Message{
		Prefix:  prefix,
		Command: irc.RPL_ENDOFNAMES,
		Params:  []string{nick, channel, "End of /NAMES list"},
	})
	return msgs
}
