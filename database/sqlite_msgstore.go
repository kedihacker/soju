package database

import (
	"database/sql"

	"git.sr.ht/~sircmpwn/go-bare"

	"git.sr.ht/~emersion/soju/database"
)

type dbMsgID struct {
	ID bare.Uint
}

func (dbMsgID) msgIDType() msgIDType {
	return msgIDDB
}

func parseDBMsgID(s string) (msgID int64, err error) {
	var id dbMsgID
	_, _, err = ParseMsgID(s, &id)
	if err != nil {
		return 0, err
	}
	return int64(id.ID), nil
}

func formatDBMsgID(netID int64, target string, msgID int64) string {
	id := dbMsgID{bare.Uint(msgID)}
	return formatMsgID(netID, target, &id)
}

type sqliteMsgStore struct {
	db *sql.DB
	userID int64
}

func (ms *sqliteMsgStore) Close() error {
	return nil
}

func (ms *sqliteMsgStore) LastMsgID(network *database.Network, entity string, t time.Time) (string, error) {
	ctx := context.TODO()
	ctx, cancel := context.WithTimeout(ctx, sqliteQueryTimeout)
	defer cancel()

	var msgID int64
	row := db.db.QueryRowContext(ctx, `
		SELECT id
		FROM Message
		WHERE network = ? AND target = ?
		ORDER BY id DESC LIMIT 1
		`, network.ID, entity)
	if err := row.Scan(&msgID); err != nil {
		return "", err
	}

	return formatDBMsgID(network.ID, entity, msgID), nil
}

func (ms *sqliteMsgStore) LoadLatestID(ctx context.Context, id string, options *msgstore.LoadMessageOptions) ([]*irc.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, sqliteQueryTimeout)
	defer cancel()

	rows, err := db.db.QueryContext(ctx, `
		SELECT raw
		FROM Message
		WHERE network = ? AND target = ? AND
			command IN ("PRIVMSG", "NOTICE", "TAGMSG")
		ORDER BY id DESC LIMIT ?`,
		options.Network.ID, options.Entity, options.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var l []*irc.Message
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}

		msg, err := irc.ParseMessage(raw)
		if err != nil {
			return nil, err
		}

		l = append(l, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// TODO: reverse list
	return l, nil
}

func (ms *sqliteMsgStore) Append(network *database.Network, entity string, msg *irc.Message) (id string, err error) {
	ctx := context.TODO()
	ctx, cancel := context.WithTimeout(ctx, sqliteQueryTimeout)
	defer cancel()

	var t time.Time
	if tag, ok := msg.Tags["time"]; ok {
		var err error
		t, err = time.Parse(xirc.ServerTimeLayout, string(tag))
		if err != nil {
			return "", fmt.Errorf("failed to parse message time tag: %v", err)
		}
	} else {
		t = time.Now()
	}

	res, err := db.db.ExecContext(ctx, `
		INSERT INTO Message(network, target, raw, command, time)
		VALUES (:network, :target, :raw, :command, :time, :target)`,
		sql.Named("network", network.ID),
		sql.Named("target", sql.NullString{String: entity, Valid: entity != ""}),
		sql.Named("raw", msg.String()),
		sql.Named("command", msg.Command),
		sql.Named("time", formatSqliteTime(t)),
	)
	if err != nil {
		return "", err
	}
	msgID, err := res.LastInsertId()
	if err != nil {
		return "", err
	}

	return formatDBMsgID(network.ID, entity, msgID), nil
}
