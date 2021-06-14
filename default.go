package zord

import "github.com/rs/zerolog"

func DefaultFirstKeys() []string {
	return []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		zerolog.CallerFieldName,
		zerolog.ErrorFieldName,
		zerolog.MessageFieldName,
	}
}
