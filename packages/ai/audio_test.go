package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/ai"
)

func TestFakeTTSAndSTT(t *testing.T) {
	m := ai.New()
	tts, err := m.TextToSpeech(context.Background(), ai.TTSRequest{Input: "hello"})
	if err != nil || !bytes.Contains(tts.Audio, []byte("hello")) {
		t.Fatalf("%v %+v", err, tts)
	}
	stt, err := m.SpeechToText(context.Background(), ai.STTRequest{Audio: []byte("wav")})
	if err != nil || !strings.Contains(stt.Text, "transcript") {
		t.Fatalf("%v %+v", err, stt)
	}
	if !m.Supports(ai.CapSpeech, "fake") {
		t.Fatal("cap")
	}
}

func TestOpenAITTS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			t.Fatalf("%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("MP3DATA"))
	}))
	defer srv.Close()
	m := ai.New()
	m.Extend("oa", &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", APIKey: "k", HTTPClient: srv.Client()})
	resp, err := m.Using("oa").TextToSpeech(context.Background(), ai.TTSRequest{Input: "hi", Voice: "alloy"})
	if err != nil || string(resp.Audio) != "MP3DATA" {
		t.Fatalf("%v %+v", err, resp)
	}
}

func TestOpenAISTT(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Fatalf("%s", r.URL.Path)
		}
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
			t.Fatalf("%v %s", err, mediaType)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		found := false
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if part.FormName() == "file" {
				found = true
			}
			_, _ = io.Copy(io.Discard, part)
		}
		if !found {
			t.Fatal("no file")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "hello world"})
	}))
	defer srv.Close()
	m := ai.New()
	m.Extend("oa", &ai.OpenAIDriver{BaseURL: srv.URL + "/v1", APIKey: "k", HTTPClient: srv.Client()})
	resp, err := m.Using("oa").SpeechToText(context.Background(), ai.STTRequest{Audio: []byte("abc"), Filename: "a.mp3"})
	if err != nil || resp.Text != "hello world" {
		t.Fatalf("%v %+v", err, resp)
	}
}
