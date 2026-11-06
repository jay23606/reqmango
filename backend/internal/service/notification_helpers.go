package service

// filterRecipientsForSender removes the actor from recipient lists so business
// events never enqueue a self-notification (FR-05).
func filterRecipientsForSender(recipientIDs []uint64, senderID *uint64) []uint64 {
	if senderID == nil || *senderID == 0 {
		return recipientIDs
	}
	actor := *senderID
	out := recipientIDs[:0]
	for _, id := range recipientIDs {
		if id != actor {
			out = append(out, id)
		}
	}
	return out
}

func skipSelfNotify(recipientID uint64, senderID *uint64) bool {
	return senderID != nil && *senderID != 0 && recipientID == *senderID
}
