package gemini

// SendProductImageFunc sends a product image via WhatsApp. Registered by the
// whatsapp package at init to avoid an import cycle (gemini ↔ whatsapp).
var SendProductImageFunc func(deviceID uint, jid, imageURL, caption string) error
