package zord

import "github.com/rs/zerolog"

const defaultMaxDepth int = 64

func DefaultFirstKeys() []string {
	return []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		zerolog.CallerFieldName,
		zerolog.ErrorFieldName,
		zerolog.MessageFieldName,
	}
}
