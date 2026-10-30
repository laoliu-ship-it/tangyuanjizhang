package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"fandianjizhang/server/config"
	"fandianjizhang/server/internal/dto"
	"fandianjizhang/server/internal/middleware"
	"fandianjizhang/server/internal/model"

	"github.com/gin-gonic/gin"
)

// ============================================================
// Upload 去重逻辑：media_file 命中但没有 ocr_record 时必须补跑 OCR，
// 不能返回 ocr_id=0 / raw_texts 为空的「成功」响应。
// 线上现象：iPhone 上传 jpeg 较慢，前端 25s 超时中断后 media_file 已落库、
// ocr_record 未写入，用户点重试永远拿到空结果。
// ============================================================

type fakeMediaFileRepo struct {
	existing *model.MediaFile
	created  []*model.MediaFile
}

func (r *fakeMediaFileRepo) GetByHash(_ context.Context, _ uint64, _ string) (*model.MediaFile, error) {
	return r.existing, nil
}

func (r *fakeMediaFileRepo) Create(_ context.Context, mf *model.MediaFile) error {
	r.created = append(r.created, mf)
	return nil
}

type fakeOcrRecordRepo struct {
	existing *model.OcrRecord
	created  []*model.OcrRecord
}

func (r *fakeOcrRecordRepo) Create(_ context.Context, rec *model.OcrRecord) error {
	r.created = append(r.created, rec)
	return nil
}

func (r *fakeOcrRecordRepo) GetByID(_ context.Context, id, _ uint64) (*model.OcrRecord, error) {
	return r.existing, nil
}

func (r *fakeOcrRecordRepo) GetByHash(_ context.Context, _ uint64, _ string) (*model.OcrRecord, error) {
	return r.existing, nil
}

type fakeOCRService struct {
	calls int
	last  string
}

func (s *fakeOCRService) ProcessImage(_ context.Context, imagePath string) (*dto.OCRResult, error) {
	s.calls++
	s.last = imagePath
	return &dto.OCRResult{
		RawTexts: []dto.OCRText{{Text: "螺丝 -15.00", Confidence: 1}},
	}, nil
}

type fakeStorage struct {
	saved []string
}

func (s *fakeStorage) Save(_ uint64, date, filename string, _ io.Reader) (string, error) {
	s.saved = append(s.saved, date+"/"+filename)
	return "uploads/2/" + date + "/" + filename, nil
}

type fakeCreditsService struct {
	actions []string
}

func (s *fakeCreditsService) GetBalance(context.Context, uint64) (*dto.CreditsResp, error) {
	return nil, nil
}
func (s *fakeCreditsService) CheckAndDeduct(_ context.Context, _ uint64, action, _ string, _ uint64, _ string) error {
	s.actions = append(s.actions, action)
	return nil
}
func (s *fakeCreditsService) Gift(context.Context, uint64, int64, string, string) error { return nil }
func (s *fakeCreditsService) ExpireDueCredits(context.Context) (int64, error)           { return 0, nil }
func (s *fakeCreditsService) SetUnlimited(context.Context, uint64, bool) error          { return nil }
func (s *fakeCreditsService) GetLogs(context.Context, uint64, int, int, string) (*dto.CreditLogsPageResp, error) {
	return nil, nil
}
func (s *fakeCreditsService) GetTenantCredits(context.Context, uint64) (*dto.TenantCreditsDetail, error) {
	return nil, nil
}
func (s *fakeCreditsService) ListTenantCredits(context.Context, []uint64) ([]*dto.TenantCreditsDetail, error) {
	return nil, nil
}

// newUploadContext 构造带 tenant 上下文与 multipart 文件的 gin 上下文
func newUploadContext(t *testing.T, hash string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "IMG_0001.jpg")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("\xff\xd8\xff\xe0 fake jpeg bytes")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if hash != "" {
		_ = w.WriteField("original_hash", hash)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/api/upload/ocr", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.Request = req
	c.Set(middleware.ContextKeyTenantID, uint64(2))
	return c, rec
}

func newTestHandler(media *fakeMediaFileRepo, ocrRepo *fakeOcrRecordRepo, ocrSvc *fakeOCRService, credits *fakeCreditsService) *OCRHandler {
	return &OCRHandler{
		ocrSvc:        ocrSvc,
		creditsSvc:    credits,
		storage:       &fakeStorage{},
		mediaRepo:     media,
		ocrRecordRepo: ocrRepo,
		uploadCfg:     config.UploadConfig{MaxSize: 1024 * 1024},
	}
}

func decodeOCRResponse(t *testing.T, rec *httptest.ResponseRecorder) dto.OCRResponse {
	t.Helper()
	var resp struct {
		Code int             `json:"code"`
		Data dto.OCRResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response %q: %v", rec.Body.String(), err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d, body=%s", resp.Code, rec.Body.String())
	}
	return resp.Data
}

// 命中 media_file 但没有 ocr_record（上次 OCR 中断）→ 必须补跑 OCR 并落库、扣费
func TestUpload_MediaHitWithoutOcrRecord_ReRunsOCR(t *testing.T) {
	media := &fakeMediaFileRepo{existing: &model.MediaFile{
		ID:           7,
		TenantID:     2,
		OriginalHash: "hash-1",
		FileName:     "IMG_0001.jpg",
		FilePath:     "uploads/2/2026-10-30/111_ocr.jpg",
		FileSize:     12345,
	}}
	ocrRepo := &fakeOcrRecordRepo{}
	ocrSvc := &fakeOCRService{}
	credits := &fakeCreditsService{}
	h := newTestHandler(media, ocrRepo, ocrSvc, credits)

	c, rec := newUploadContext(t, "hash-1")
	h.Upload(c)

	data := decodeOCRResponse(t, rec)
	if ocrSvc.calls != 1 {
		t.Fatalf("expected OCR to be re-run once, got %d", ocrSvc.calls)
	}
	if ocrSvc.last != media.existing.FilePath {
		t.Errorf("expected OCR on existing file %q, got %q", media.existing.FilePath, ocrSvc.last)
	}
	if len(data.RawTexts) == 0 {
		t.Fatalf("expected non-empty raw_texts, got %+v", data)
	}
	if len(ocrRepo.created) != 1 {
		t.Fatalf("expected ocr_record created once, got %d", len(ocrRepo.created))
	}
	if ocrRepo.created[0].ImageHash != "hash-1" {
		t.Errorf("expected ocr_record bound to hash-1, got %q", ocrRepo.created[0].ImageHash)
	}
	if len(media.created) != 0 {
		t.Errorf("expected no new media_file, got %d", len(media.created))
	}
	if len(credits.actions) != 2 {
		t.Errorf("expected upload+ocr deducted, got %v", credits.actions)
	}
}

// 命中 media_file 且有 ocr_record → 直接复用，不跑 OCR、不扣费
func TestUpload_DuplicateWithOcrRecord_ReusesResult(t *testing.T) {
	media := &fakeMediaFileRepo{existing: &model.MediaFile{
		ID:       7,
		FileName: "IMG_0001.jpg",
		FilePath: "uploads/2/2026-10-30/111_ocr.webp",
	}}
	ocrRepo := &fakeOcrRecordRepo{existing: &model.OcrRecord{
		ID:        42,
		Amount:    15,
		Date:      "2026-10-15",
		RawTexts:  `["螺丝 -15.00"]`,
		AIMode:    true,
		ImageHash: "hash-1",
	}}
	ocrSvc := &fakeOCRService{}
	credits := &fakeCreditsService{}
	h := newTestHandler(media, ocrRepo, ocrSvc, credits)

	c, rec := newUploadContext(t, "hash-1")
	h.Upload(c)

	data := decodeOCRResponse(t, rec)
	if ocrSvc.calls != 0 {
		t.Fatalf("expected cached OCR result, got %d OCR calls", ocrSvc.calls)
	}
	if len(credits.actions) != 0 {
		t.Errorf("expected no deduction for duplicate, got %v", credits.actions)
	}
	if data.OcrID != 42 || !data.Duplicate {
		t.Errorf("expected reused ocr_id 42 with duplicate=true, got %+v", data)
	}
	if len(data.RawTexts) != 1 || data.RawTexts[0] != "螺丝 -15.00" {
		t.Errorf("expected cached raw_texts, got %v", data.RawTexts)
	}
}

// 全新文件 → 存盘、落 media_file、跑 OCR、扣费
func TestUpload_NewFile_SavesAndRunsOCR(t *testing.T) {
	media := &fakeMediaFileRepo{}
	ocrRepo := &fakeOcrRecordRepo{}
	ocrSvc := &fakeOCRService{}
	credits := &fakeCreditsService{}
	h := newTestHandler(media, ocrRepo, ocrSvc, credits)

	c, rec := newUploadContext(t, "hash-new")
	h.Upload(c)

	data := decodeOCRResponse(t, rec)
	if ocrSvc.calls != 1 {
		t.Fatalf("expected one OCR call, got %d", ocrSvc.calls)
	}
	if len(media.created) != 1 || media.created[0].OriginalHash != "hash-new" {
		t.Fatalf("expected media_file created for hash-new, got %+v", media.created)
	}
	if len(ocrRepo.created) != 1 || len(data.RawTexts) == 0 {
		t.Fatalf("expected ocr_record and non-empty raw_texts, got %+v", data)
	}
	if len(credits.actions) != 2 {
		t.Errorf("expected upload+ocr deducted, got %v", credits.actions)
	}
}
