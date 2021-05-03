package log_observer_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLogObserver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LogObserver Suite")
}
