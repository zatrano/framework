// Package mailqueue — e-posta gönderimini asenkron yapar.
// Sync SMTP çağrıları işleyiciyi bloklar; bu paket goroutine tabanlı
// bellek içi kuyruk ile mail'leri arka planda gönderir.
// Production için aynı arayüzü Redis/NATS tabanlı kuyruk ile değiştirebilirsiniz.
package mailqueue

import (
	"context"
	"sync"
	"time"

	"github.com/zatrano/framework/configs/logconfig"

	"go.uber.org/zap"
)

// MailJob — kuyruklanacak e-posta işi.
type MailJob struct {
	To       string
	Subject  string
	Body     string
	Template string
	Data     map[string]interface{}
	IsHTML   bool
	Attempt  int
	MaxRetry int
}

// Sender — mail gönderen arayüzü (IMailService'i wrap eder).
type Sender interface {
	SendMail(to, subject, body string) error
	SendTemplateMail(to, subject, tmplName string, data map[string]interface{}) error
}

// Kuyruk — asenkron e-posta kuyruğu.
type Queue struct {
	jobs    chan MailJob
	sender  Sender
	wg      sync.WaitGroup
	workers int
	done    chan struct{}
}

var (
	defaultQueue *Queue
	once         sync.Once
)

// Init — global kuyruğu başlatır. main() içinde çağrılmalıdır.
func Init(sender Sender, workers int, bufferSize int) {
	once.Do(func() {
		if workers <= 0 {
			workers = 3
		}
		if bufferSize <= 0 {
			bufferSize = 200
		}
		defaultQueue = &Queue{
			jobs:    make(chan MailJob, bufferSize),
			sender:  sender,
			workers: workers,
			done:    make(chan struct{}),
		}
		defaultQueue.start()
		logconfig.SLog.Infow("Mail queue başlatıldı",
			"workers", workers,
			"buffer_size", bufferSize)
	})
}

// Shutdown — worker'ları graceful kapatır. main() defer'ında çağrılmalıdır.
func Shutdown(ctx context.Context) {
	if defaultQueue == nil {
		return
	}
	close(defaultQueue.done)

	done := make(chan struct{})
	go func() {
		defaultQueue.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logconfig.SLog.Info("Mail queue kapatıldı")
	case <-ctx.Done():
		logconfig.SLog.Warn("Mail queue kapatma timeout")
	}
}

// Enqueue — e-posta işini kuyruğa ekler. Non-blocking; kuyruk doluysa warn log atar.
func Enqueue(job MailJob) {
	if defaultQueue == nil {
		logconfig.Log.Warn("Mail queue başlatılmamış, senkron gönderime geçiliyor",
			zap.String("to", job.To))
		return
	}
	if job.MaxRetry == 0 {
		job.MaxRetry = 3
	}
	select {
	case defaultQueue.jobs <- job:
	default:
		logconfig.Log.Warn("Mail queue dolu, iş düşürüldü",
			zap.String("to", job.To),
			zap.String("subject", job.Subject))
	}
}

// Send — template mail'i kuyruğa ekler (kısa yol).
func Send(to, subject, template string, data map[string]interface{}) {
	Enqueue(MailJob{
		To:       to,
		Subject:  subject,
		Template: template,
		Data:     data,
		MaxRetry: 3,
	})
}

// SendPlain — düz metin/HTML mail'i kuyruğa ekler (kısa yol).
func SendPlain(to, subject, body string) {
	Enqueue(MailJob{
		To:       to,
		Subject:  subject,
		Body:     body,
		IsHTML:   true,
		MaxRetry: 3,
	})
}

func (q *Queue) start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

func (q *Queue) worker(id int) {
	defer q.wg.Done()
	logconfig.SLog.Debugw("Mail worker başladı", "worker_id", id)

	for {
		select {
		case <-q.done:
			// Kuyrukta kalan işleri bitir
			for {
				select {
				case job := <-q.jobs:
					q.process(job, id)
				default:
					return
				}
			}
		case job := <-q.jobs:
			q.process(job, id)
		}
	}
}

func (q *Queue) process(job MailJob, workerID int) {
	job.Attempt++

	var err error
	if job.Template != "" {
		err = q.sender.SendTemplateMail(job.To, job.Subject, job.Template, job.Data)
	} else {
		err = q.sender.SendMail(job.To, job.Subject, job.Body)
	}

	if err == nil {
		logconfig.Log.Info("Mail gönderildi",
			zap.String("to", job.To),
			zap.String("subject", job.Subject),
			zap.Int("worker_id", workerID),
			zap.Int("attempt", job.Attempt))
		return
	}

	logconfig.Log.Warn("Mail gönderilemedi",
		zap.String("to", job.To),
		zap.String("subject", job.Subject),
		zap.Int("attempt", job.Attempt),
		zap.Int("max_retry", job.MaxRetry),
		zap.Error(err))

	if job.Attempt < job.MaxRetry {
		backoff := time.Duration(job.Attempt*job.Attempt) * time.Second
		time.Sleep(backoff)
		select {
		case q.jobs <- job:
		default:
			logconfig.Log.Error("Retry kuyruğa eklenemedi, iş kalıcı olarak düşürüldü",
				zap.String("to", job.To))
		}
	} else {
		logconfig.Log.Error("Mail maksimum deneme sayısına ulaştı, kalıcı hata",
			zap.String("to", job.To),
			zap.String("subject", job.Subject),
			zap.Int("attempts", job.Attempt))
	}
}
