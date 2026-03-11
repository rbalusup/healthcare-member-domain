package member_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rbalusup/healthcare-member-domain/internal/domain/member"
)

func TestSentinelErrors_AreDistinct(t *testing.T) {
	errs := []error{
		member.ErrMemberNotFound,
		member.ErrMemberAlreadyExists,
		member.ErrInvalidMemberData,
		member.ErrMemberEnrollmentFailed,
		member.ErrMemberNotActive,
		member.ErrInvalidRiskScore,
	}

	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			assert.NotEqual(t, errs[i], errs[j],
				"errors at index %d and %d should be distinct", i, j)
		}
	}
}

func TestSentinelErrors_HaveMessages(t *testing.T) {
	assert.NotEmpty(t, member.ErrMemberNotFound.Error())
	assert.NotEmpty(t, member.ErrMemberAlreadyExists.Error())
	assert.NotEmpty(t, member.ErrInvalidMemberData.Error())
	assert.NotEmpty(t, member.ErrMemberEnrollmentFailed.Error())
	assert.NotEmpty(t, member.ErrMemberNotActive.Error())
	assert.NotEmpty(t, member.ErrInvalidRiskScore.Error())
}
