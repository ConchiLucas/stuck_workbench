package plan

import (
	"context"
	"errors"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestParseHandwritingSnapshot(t *testing.T) {
	raw := `{"schemaVersion":2,"subjectCode":"literacy","kpId":1,"questionType":"write_char","skillCode":"write_char","interaction":"handwriting","responseSchemaVersion":2,"evaluationPolicyVersion":"ink-match-v1","stem":{"audio":{"revisionId":"abc","kind":"speech","sha256":"abc"}},"writingTemplate":{"revisionId":"abc","kind":"writing_template","sha256":"abc"}}`
	s, err := ParseSnapshot(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Interaction != "handwriting" {
		t.Fatal(s)
	}
}

func TestClaimRejectsCorruptCachedTemplateBeforeCreatingPlan(t *testing.T) {
	d, e := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = d.Exec("CREATE TABLE question_versions(revision_id INTEGER,snapshot_json TEXT)").Error; e != nil {
		t.Fatal(e)
	}
	if e = d.AutoMigrate(&CachedTemplate{}); e != nil {
		t.Fatal(e)
	}
	raw := `{"schemaVersion":2,"subjectCode":"literacy","kpId":1,"targetText":"一","questionType":"write_char","skillCode":"write_char","interaction":"handwriting","responseSchemaVersion":2,"evaluationPolicyVersion":"ink-match-v1","stem":{"audio":{"revisionId":"rev","kind":"speech"}},"writingTemplate":{"revisionId":"rev","kind":"writing_template","sha256":"hash"}}`
	d.Exec("INSERT INTO question_versions VALUES(1,?)", raw)
	d.Create(&CachedTemplate{RevisionID: "rev", SHA256: "hash", TemplateJSON: "{broken"})
	if e = NewService(d).cacheRevisionTemplates(context.Background(), 1); !errors.Is(e, ErrTemplateUnavailable) {
		t.Fatalf("cache corruption must fail before plan creation: %v", e)
	}
}
