package emailable

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
)

const base_url = "https://api.emailable.com/v1"
const api_key_param = "api_key"

type Emailable struct {
	apiKey string
}

func NewEmailable(apiKey string) Emailable {
	return Emailable{
		apiKey: apiKey,
	}
}

func (emailable *Emailable) Verify(req VerifyRequest) (*VerifyResponse, error) {
	var vresp VerifyResponse
	url, err := url.Parse(fmt.Sprintf("%s/verify", base_url))
	if err != nil {
		return nil, err
	}
	var timeout uint8
	if req.Timeout < 5 {
		timeout = 5
	} else if req.Timeout > 30 {
		timeout = 30
	} else {
		timeout = req.Timeout
	}
	values := url.Query()
	values.Add("email", req.Email)
	values.Add("smtp", fmt.Sprint(req.Smtp))
	values.Add("timeout", fmt.Sprint(timeout))
	values.Add(api_key_param, emailable.apiKey)
	url.RawQuery = values.Encode()
	hresp, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}
	if hresp.StatusCode == 249 {
		return nil, errors.New("your request is taking longer than normal. please send your request again")
	} else if hresp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %v", hresp.StatusCode)
	}
	err = json.NewDecoder(hresp.Body).Decode(&vresp)
	if err != nil {
		return nil, err
	}
	return &vresp, nil
}

func (emailable *Emailable) BatchVerify(req BatchRequest) (*BatchResponse, error) {
	var batch BatchResponse
	if len(req.Emails) > 1000 {
		return nil, fmt.Errorf("too much email given")
	}
	value, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(fmt.Sprintf("%s/batch", base_url), "application/json", bytes.NewReader(value))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&batch)
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (emailable *Emailable) BatchFromFile(file io.Reader) ([]BatchResponse, error) {
	const maxSize = 999
	var results []BatchResponse
	scanner := bufio.NewScanner(file)
	var buffer []string = make([]string, 0, maxSize)

	for scanner.Scan() {
		if len(buffer) == maxSize {
			resp, err := emailable.BatchVerify(emailable.NewBatchReq(buffer))
			if err != nil {
				return nil, err
			}
			results = append(results, *resp)
			buffer = nil
		}
		email := scanner.Text()
		_, err := mail.ParseAddress(email)
		if err == nil {
			buffer = append(buffer, email)
		}
	}
	if len(buffer) > 0 {
		resp, err := emailable.BatchVerify(emailable.NewBatchReq(buffer))
		if err == nil {
			results = append(results, *resp)
		}
	}
	return results, nil
}

func (emailable *Emailable) BatchStatus(req *BatchStatusRequest) (*BatchStatusResponse, error) {
	var resp BatchStatusResponse
	url, err := url.Parse(fmt.Sprintf("%s/batch", base_url))
	if err != nil {
		return nil, err
	}
	querys := url.Query()
	querys.Add(api_key_param, req.ApiKey)
	querys.Add("id", req.Id)
	if req.Partial != "" {
		querys.Add("partial", req.Partial)
	}
	if req.Smiluate != "" {
		querys.Add("simulate", req.Smiluate)
	}
	url.RawQuery = querys.Encode()
	hresp, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(hresp.Body).Decode(&resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type VerifyRequest struct {
	Email     string `json:"email"`      // The email you want verified.
	Smtp      bool   `json:"smtp"`       // "true" or "false". The SMTP step takes up a majority of the API's response time. If you would like to speed up your response times, you can disable this step. Default: true
	AcceptAll bool   `json:"accept_all"` // "true" or "false". Whether or not an accept-all check is performed. Heavily impacts API's response time. Default: false
	Timeout   uint8  `json:"timeout"`    // Optional timeout to wait for response (in seconds). Min: 5, Max: 30. Default: 5
	ApiKey    string `json:"api_key"`
}

type VerifyResponse struct {
	AcceptAll    bool    `json:"accept_all"`    // Whether the mail server used to verify indicates that all addresses are deliverable regardless of whether or not the email is valid.
	DidYouMean   string  `json:"did_you_mean"`  // A suggested correction for a common misspelling.
	Disposable   bool    `json:"disposable"`    // Whether this email is hosted on a disposable or temporary email service.
	Domain       string  `json:"domain"`        // The domain of the email.
	Duration     float64 `json:"duration"`      // The length of time (in seconds) spent verifying this email.
	Email        string  `json:"email"`         // The email that was verified.
	FirstName    string  `json:"first_name"`    // The possible first name of the user.
	Free         bool    `json:"free"`          // Whether the email is hosted by a free email provider.
	FullName     string  `json:"full_name"`     // The possible full name of the user.
	Gender       string  `json:"gender"`        // The possible gender of the user.
	LastName     string  `json:"last_name"`     // The possible last name of the user.
	MxRecord     string  `json:"mx_record"`     // The address of the mail server used to verify the email.
	Reason       string  `json:"reason"`        // The reason for the associated state.
	Role         bool    `json:"role"`          // Whether the email is considered a role address.
	Score        int32   `json:"score"`         // The score of the verified email.
	SmtpProvider string  `json:"smtp_provider"` // The SMTP provider of the verified email's domain.
	State        string  `json:"state"`         // The state of the verified email
	Tag          string  `json:"tag"`           // The tag part of the verified email.
	User         string  `json:"user"`          // The user part of the verified email.
}

func (emailable *Emailable) NewVerifyReq(email string) VerifyRequest {
	return VerifyRequest{
		Email:     email,
		Smtp:      false,
		AcceptAll: false,
		Timeout:   5,
		ApiKey:    emailable.apiKey,
	}
}

func (emailable *Emailable) NewBatchReq(emails []string) BatchRequest {
	return BatchRequest{
		Emails:   emails,
		Url:      "",
		Retries:  false,
		Simulate: "",
		ApiKey:   emailable.apiKey,
	}
}

func (emailable *Emailable) NewBatchVerificationReq(id string) BatchStatusRequest {
	return BatchStatusRequest{
		Id:       id,
		ApiKey:   emailable.apiKey,
		Partial:  "",
		Smiluate: "",
	}
}

type SimulateValue string

const (
	GenericError             SimulateValue = "generic_error"
	InsufficientCreditsError SimulateValue = "insufficient_credits_error"
	PaymentError             SimulateValue = "payment_error"
	CardError                SimulateValue = "card_error"
)

type BatchRequest struct {
	Emails   []string      `json:"emails"`  // A comma separated list of emails.
	Url      string        `json:"url"`     // A URL that will receive the batch results via HTTP POST.
	Retries  bool          `json:"retries"` // Defaults to true. Retries increase accuracy by automatically retrying verification when our system receives certain responses from mail servers.
	Simulate SimulateValue `json:"string"`  // Used to simulate certain responses from the API while using a test key.
	ApiKey   string        `json:"api_key"`
}

type BatchResponse struct {
	Message string `json:"message"` // A message about your batch.
	Id      string `json:"id"`      // The unique ID of the batch.
}

type BatchStatusRequest struct {
	Id       string `json:"id"`
	ApiKey   string `json:"api_key"`
	Partial  string `json:"partial"`
	Smiluate string `json:"simulate"`
}

type ReasonCount struct {
	AcceptedEmail     int64 `json:"accepted_email"`
	InvalidDomain     int64 `json:"invalid_domain"`
	InvalidEmail      int64 `json:"invalid_email"`
	InvalidSmtp       int64 `json:"invalid_smtp"`
	LowDeliverability int64 `json:"low_deliverability"`
	LowQuality        int64 `json:"low_quality"`
	NoConnect         int64 `json:"no_connect"`
	RejectedEmail     int64 `json:"rejected_email"`
	Timeout           int64 `json:"timeout"`
	UnavailableSmtp   int64 `json:"unavailable_smtp"`
	UnexpectedError   int64 `json:"unexpected_error"`
}

type TotalCounts struct {
	Deliverable   int64 `json:"deliverable"`
	Processed     int64 `json:"processed"`
	Risky         int64 `json:"risky"`
	Total         int64 `json:"total"`
	Undeliverable int64 `json:"undeliverable"`
	Unknown       int64 `json:"unknown"`
}

type BatchStatusResponse struct {
	Message     string           `json:"message"`
	Processed   int64            `json:"processed"`
	Total       int64            `json:"total"`
	Emails      []VerifyResponse `json:"emails"`
	Id          string           `json:"id"`
	ReasonCount ReasonCount      `json:"reason_counts"`
	TotalCounts TotalCounts      `json:"total_counts"`
}
