package auth

import (
	"context"
	"errors"
	"strings"

	// Internal
	"github.com/revirathya/revirathya-finance-core/internal/domain"
	errDom "github.com/revirathya/revirathya-finance-core/internal/domain/error"
)

type AuthUsecase struct {
	gVeri   domain.GoogleVerifier
	usrRepo domain.UserRepository
}

func NewAuthUsecase(gVeri domain.GoogleVerifier, usrRepo domain.UserRepository) *AuthUsecase {
	return &AuthUsecase{
		gVeri:   gVeri,
		usrRepo: usrRepo,
	}
}

func (u *AuthUsecase) LoginWithGoogle(ctx context.Context, idToken string) (*domain.User, error) {
	// Verify User
	userGoog, err := u.gVeri.VerifyIdentity(ctx, idToken)
	if err != nil {
		return nil, err
	}

	if !userGoog.EmailVerified {
		return nil, errDom.ErrUnauthenticated
	}

	// Check User
	user, err := u.usrRepo.GetUserByGoogleSub(ctx, userGoog.Subject)
	if err != nil {
		if !errors.Is(err, errDom.ErrNotFound) {
			return nil, err
		} else {
			// Handle new user
			givenName := userGoog.GivenName
			if (givenName == "") && (userGoog.Email != "") {
				givenName = strings.Split(userGoog.Email, "@")[0]
			}

			var familyName *string
			if userGoog.FamilyName != "" {
				familyName = &userGoog.FamilyName
			}

			form := domain.UserForm{
				FirstName: givenName,
				LastName:  familyName,
				Email:     userGoog.Email,
				GoogleSub: userGoog.Subject,
			}

			if err = form.Validate(); err != nil {
				return nil, err
			}
			user, err = u.usrRepo.CreateUser(ctx, form)
			if err != nil {
				if errors.Is(err, errDom.ErrConflict) {
					// Handle register race-condition (fetch if already created by other ops)
					user, err = u.usrRepo.GetUserByGoogleSub(ctx, userGoog.Subject)
					if err != nil {
						if errors.Is(err, errDom.ErrNotFound) {
							return nil, errDom.ErrConflict
						}
						return nil, err
					}
					return user, nil
				}
				return nil, err
			}
		}
	}
	return user, nil
}
