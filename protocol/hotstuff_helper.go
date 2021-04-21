package protocol

type HotStuffHelper interface {
	// DiscardAboveHeight Delete blocks data greater than the baseHeight
	DiscardAboveHeight(baseHeight int64)
}
