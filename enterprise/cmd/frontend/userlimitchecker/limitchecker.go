package userlimitchecker

import (
	"context"
	"log"
	"time"

	ps "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/dotcom/productsubscription"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// send email to site admins if approaching user limit on active license
func sendApproachingUserLimitAlert(ctx context.Context, db database.DB) error {
	licenseDb := ps.NewDbLicense(db)
	licenses, err := licenseDb.List(ctx, ps.DbLicencesListNoOpt())
	if err != nil {
		return errors.Wrap(err, "could not get list of db licenses")
	}

	// find current license
	var licenseID string
	for _, license := range licenses {
		if license.RevokedAt == nil {
			licenseID = license.ID
			break
		}
	}

	// find last time alert was sent
	userCountAlertSent, err := licenseDb.GetUserCountAlertSentAt(ctx, licenseID)
	if err != nil {
		return errors.Wrap(err, "could not get last time user account alert was sent")
	}

	// return if the time right now is 7 days
	if !time.Now().UTC().After(userCountAlertSent.UTC().Add(7 * 24 * time.Hour)) {
		log.Println("email recently sent")
		return nil
	}

	userCount, err := getUserCount(ctx, db)
	if err != nil {
		return errors.Wrap(err, "could not get user count")
	}

	userLimit, err := getLicenseUserLimit(ctx, db)
	if err != nil {
		return errors.Wrap(err, "could not get license user limit")
	}

	percentOfLimit := getPercentOfLimit(userCount, userLimit)
	if percentOfLimit >= 90 || userCount >= userLimit-2 {
		siteAdminEmails, err := getSiteAdminEmails(ctx, db)
		if err != nil {
			return errors.Wrap(err, "could not get site admins")
		}

		messageId := "approaching_user_limit"
		replyTo := "support@sourcegraph.com"

		if err := txemail.Send(ctx, "approaching_user_limit", txemail.Message{
			To:        siteAdminEmails,
			Template:  approachingUserLimitEmailTemplate,
			MessageID: &messageId,
			ReplyTo:   &replyTo,
			Data: struct {
				RemainingUsers int
				Percent        int
			}{
				RemainingUsers: userLimit - userCount,
				Percent:        percentOfLimit,
			},
		}); err != nil {
			return errors.Wrap(err, "could not send email")
		} else {
			log.Println("User count on license within limit")
			return nil
		}
	}

	return nil
}

func getPercentOfLimit(userCount, userLimit int) int {
	if userCount == 0 {
		return 0
	}

	// RESEARCH: will this ever be the case?
	if userLimit == 0 {
		return userCount + 100
	}

	return (userCount * 100) / userLimit
}

func getUserCount(ctx context.Context, db database.DB) (int, error) {
	userStore := db.Users()
	userCount, err := userStore.Count(ctx, &database.UsersListOptions{})
	if err != nil {
		return 0, err
	}
	return userCount, nil
}

func getLicenseUserLimit(ctx context.Context, db database.DB) (int, error) {
	items, err := ps.NewDbLicense(db).List(ctx, ps.DbLicencesListNoOpt())
	if err != nil {
		return 0, err
	}

	for _, item := range items {
		if item.LicenseExpiresAt.UTC().After(time.Now().UTC()) {
			return *item.LicenseUserCount, nil
		}
	}

	return 0, nil
}

func getSiteAdminEmails(ctx context.Context, db database.DB) ([]string, error) {
	var siteAdminEmails []string
	users, err := db.Users().List(ctx, &database.UsersListOptions{})
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.SiteAdmin {
			email, _, err := getUserEmail(ctx, db, user)
			if err != nil {
				return nil, err
			}
			siteAdminEmails = append(siteAdminEmails, email)
		}
	}
	return siteAdminEmails, nil
}

func getUserEmail(ctx context.Context, db database.DB, u *types.User) (string, bool, error) {
	return database.UserEmailsWith(db).GetPrimaryEmail(ctx, u.ID)
}
