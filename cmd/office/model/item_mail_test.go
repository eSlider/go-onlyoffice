package model

import "testing"

func TestSubjectIsMail(t *testing.T) {
	if !SubjectIsMail(SubjectMailInbox) {
		t.Fatal("inbox should be mail")
	}
	if SubjectIsMail(SubjectProjects) {
		t.Fatal("projects should not be mail")
	}
}
