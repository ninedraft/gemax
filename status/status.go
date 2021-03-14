package status

// Code is a gemini protocol status code.
type Code int

const (
	// Undefined is a default empty status code value.
	Undefined Code = 0

	// Input means that the requested resource accepts a line of textual user input.  The <META> line is a prompt which
	// should be displayed to the user.  The same resource should then be requested again with the user's
	// input included as a query component.  Queries are included in requests as per the usual generic URL
	// definition in RFC3986, i.e. separated from the path by a ?.  Reserved characters used in the user's
	// input must be "percent-encoded" as per RFC3986, and space characters should also be percent-encoded.
	Input Code = 10

	// Success means that the request was handled successfully and a response body will follow the
	// response header.  The <META> line is a MIME media type which applies to
	// the response body.
	Success Code = 20

	// Redirect means that the server is redirecting the client to a new
	// location for the requested resource.  There is no response body.  <META>
	// is a new URL for the requested resource.  The URL may be absolute or
	// relative.  The redirect should be considered temporary, i.e. clients
	// should continue to request the resource at the original address and
	// should not performance convenience actions like automatically updating
	// bookmarks.  There is no response body.
	Redirect Code = 30

	// TemporaryFailure means that The request has failed.  There is no response
	// body.  The nature of the failure is temporary, i.e. an identical request
	// MAY succeed in the future.  The contents of <META> may provide additional
	// information on the failure, and should be displayed to human users.
	TemporaryFailure Code = 40

	// PermanentFailure means that the request has failed.  There is no response
	// body.  The nature of the failure is permanent, i.e. identical future
	// requests will reliably fail for the same reason.  The contents of <META>
	// may provide additional information on the failure, and should be
	// displayed to human users.  Automatic clients such as aggregators or
	// indexing crawlers should not repeat this request.
	PermanentFailure Code = 50

	// ClientCertificateRequired means that the requested resource requires a
	// client certificate to access.  If the request was made without a
	// certificate, it should be repeated with one.  If the request was made
	// with a certificate, the server did not accept it and the request should
	// be repeated with a different certificate.  The contents of <META> (and/or
	// the specific 6x code) may provide additional information on certificate
	// requirements or the reason a certificate was rejected.
	ClientCertificateRequired Code = 60
)

func (code Code) String() string {
	switch code.Class() {
	case Input:
		return "INPUT"
	case Success:
		return "SUCCESS"
	case Redirect:
		return "REDIRECT"
	case TemporaryFailure:
		return "TEMPORARY FAILURE"
	case PermanentFailure:
		return "PERMANENT FAILURE"
	case ClientCertificateRequired:
		return "CLIENT CERTIFICATE REQUIRED"
	default:
		return "undefined status code"
	}
}

// Class returns on of status code constants.
// That is, .Class rounds code to matchin CONST value:
// 	(code >= CONST && code < CONST_NEXT).
// Returns Undefined is Code is not valid.
//nolint:gocyclo // it's the simpliest way to round.
func (code Code) Class() Code {
	switch {
	case code >= Input && code < Success:
		return Input
	case code >= Success && code < Redirect:
		return Success
	case code >= Redirect && code < TemporaryFailure:
		return Redirect
	case code >= TemporaryFailure && code < PermanentFailure:
		return TemporaryFailure
	case code >= PermanentFailure && code < PermanentFailure+10:
		return PermanentFailure
	default:
		return Undefined
	}
}
