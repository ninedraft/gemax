package gemax

// MIMEGemtext describes a MIME type for gemini text.
// As a subtype of the top-level media type "text", "text/gemini" inherits the
// "charset" parameter defined in RFC 2046.  However, as noted in 3.3, the
// default value of "charset" is "UTF-8" for "text" content transferred via
// Gemini.
//
// A single additional parameter specific to the "text/gemini" subtype is
// defined: the "lang" parameter.  The value of "lang" denotes the natural
// language or language(s) in which the textual content of a "text/gemini"
// document is written.  The presence of the "lang" parameter is optional.  When
// the "lang" parameter is present, its interpretation is defined entirely by
// the client.  For example, clients which use text-to-speech technology to make
// Gemini content accessible to visually impaired users may use the value of
// "lang" to improve pronunciation of content.  Clients which render text to a
// screen may use the value of "lang" to determine whether text should be
// displayed left-to-right or right-to-left.  Simple clients for users who only
// read languages written left-to-right may simply ignore the value of "lang".
// When the "lang" parameter is not present, no default value should be assumed
// and clients which require some notion of a language in order to process the
// content (such as text-to-speech screen readers) should rely on user-input to
// determine how to proceed in the absence of a "lang" parameter.
//
// Valid values for the "lang" parameter are comma-separated lists of one or
// more language tags as defined in RFC4646.  For example:
//  * "text/gemini; lang=en" Denotes a text/gemini document written in English
//  * "text/gemini; lang=fr" Denotes a text/gemini document written in French
//  * "text/gemini; lang=en,fr" Denotes a text/gemini document written in a mixture of English and French
//  * "text/gemini; lang=de-CH" Denotes a text/gemini document written in Swiss German
//  * "text/gemini; lang=sr-Cyrl" Denotes a text/gemini document written in Serbian using the Cyrllic script
//  * "text/gemini; lang=zh-Hans-CN" Denotes a text/gemini document written in Chinese using
// 	the Simplified script as used in mainland China
const MIMEGemtext = "text/gemini"
