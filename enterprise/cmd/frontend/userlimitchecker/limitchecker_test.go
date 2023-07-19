package userlimitchecker

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/sourcegraph/log"
	ps "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/dotcom/productsubscription"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/license"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestSendApproachingUserLimitAlert(t *testing.T) {
	logger := log.NoOp()
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()

	userStore := db.Users()
	user, err := userStore.Create(ctx, users[0])
	if err != nil {
		t.Errorf("could not create user: %s", err)
	}

	t.Run("sends correctly formatted email", func(t *testing.T) {
		subStore := ps.NewDbSubscription(db)
		subId, err := subStore.Create(ctx, user.ID, user.Username)
		if err != nil {
			t.Errorf("could not create subscription: %s", err)
		}

		licensesStore := ps.NewDbLicense(db)
		licensesStore.Create(ctx, subId, "12345", 5, license.Info{
			UserCount: 2,
			ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
		})

		var gotEmail txemail.Message
		txemail.MockSend = func(ctx context.Context, message txemail.Message) error {
			gotEmail = message
			return nil
		}
		t.Cleanup(func() { txemail.MockSend = nil })

		err = sendApproachingUserLimitAlert(ctx, db)
		if err != nil {
			t.Errorf("could not send email: %s", err)
		}

		replyTo := "support@sourcegraph.com"
		messageId := "approaching_user_limit"
		want := &txemail.Message{
			To:        []string{"test@test.com"},
			Template:  approachingUserLimitEmailTemplate,
			MessageID: &messageId,
			ReplyTo:   &replyTo,
			Data: SetApproachingUserLimitTemplateData{
				RemainingUsers: 1,
			},
		}

		assert.Equal(t, want.To, gotEmail.To)
		assert.Equal(t, approachingUserLimitEmailTemplate, gotEmail.Template)
		assert.Equal(t, want.MessageID, gotEmail.MessageID)
		assert.Equal(t, want.ReplyTo, gotEmail.ReplyTo)
		assert.Equal(t, want.MessageID, gotEmail.MessageID)
		gotEmailData := want.Data.(SetApproachingUserLimitTemplateData)
		assert.Equal(t, 1, gotEmailData.RemainingUsers)
	})

	t.Run("does not send email if email sent within 7 days of current time", func(t *testing.T) {
		subStore := ps.NewDbSubscription(db)
		subId, err := subStore.Create(ctx, user.ID, user.Username)
		if err != nil {
			t.Errorf("could not create subscription: %s", err)
		}

		licensesStore := ps.NewDbLicense(db)
		licensesStore.Create(ctx, subId, "12345", 5, license.Info{
			UserCount: 2,
			ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
		})


		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err = sendApproachingUserLimitAlert(ctx, db)
		if err != nil {
			t.Errorf("could not run sendApproachingUserLimitAlert function: %s", err)
		}

		w.Close()
		out, _ := ioutil.ReadAll(r)
		os.Stdout = old

		if string(out) != "email recently sent\n" {
			t.Errorf("Expected 'email recently sent' to be printed, got %q", string(out))
		}
	})

	t.Run("does not send email if user count is not approaching user limit", func(t *testing.T) {
		subStore := ps.NewDbSubscription(db)
		subId, err := subStore.Create(ctx, user.ID, user.Username)
		if err != nil {
			t.Errorf("could not create subscription: %s", err)
		}

		licensesStore := ps.NewDbLicense(db)
		licensesStore.Create(ctx, subId, "12345", 10, license.Info{})

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err = sendApproachingUserLimitAlert(ctx, db)
		if err != nil {
			t.Errorf("could not run sendApproachingUserLimitAlert function")
		}

		w.Close()
		out, _ := ioutil.ReadAll(r)
		os.Stdout = old

		if string(out) != "User count on license within limit\n" {
			t.Errorf("Expected 'User count on license within limit' to be printed, but it wasn't")
		}
	})
}

func TestGetPercentOfLimit(t *testing.T) {
	cases := []struct {
		want      int
		userCount int
		userLimit int
	}{
		{want: 0, userCount: 0, userLimit: 100},
		{want: 84, userCount: 211, userLimit: 250},
		{want: 61, userCount: 348, userLimit: 567},
		{want: 46, userCount: 583, userLimit: 1264},
		{want: 110, userCount: 10, userLimit: 0},
		{want: 112, userCount: 45, userLimit: 40},
		{want: 95, userCount: 19, userLimit: 20},
		{want: 96, userCount: 87, userLimit: 90},
		{want: 95, userCount: 95, userLimit: 100},
		{want: 95, userCount: 3350, userLimit: 3500},
	}

	for _, tc := range cases {
		gotPercent := getPercentOfLimit(tc.userCount, tc.userLimit)
		if gotPercent != tc.want {
			t.Errorf("got %v want %v", gotPercent, tc.want)
		}
	}
}

func TestGetLicenseUserLimit(t *testing.T) {
	logger := log.NoOp()
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()

	// need a user to satisfy product_subscription foreign key constraint
	userStore := db.Users()
	user, err := userStore.Create(ctx, users[0])
	if err != nil {
		t.Errorf("could not create user: %s", err)
	}

	// need a product_subscription to satisfy product_license foreign key constraint
	subStore := ps.NewDbSubscription(db)
	subId, err := subStore.Create(ctx, user.ID, user.Username)
	if err != nil {
		t.Errorf("could not create subscription: %s", err)
	}

	licensesStore := ps.NewDbLicense(db)
	for _, license := range licensesToCreate {
		_, err = licensesStore.Create(
			ctx,
			subId,
			license.licenseId,
			license.version,
			license.licenseInfo,
		)
		if err != nil {
			t.Errorf("could not create license:, %s", err)
		}
	}

	got, err := getLicenseUserLimit(ctx, db)
	if err != nil {
		t.Errorf("could not get user limit: %s", err)
	}

	want := 30
	if got != want {
		t.Errorf("got %d want %d", got, want)
	}
}

func TestGetUserCount(t *testing.T) {
	logger := log.NoOp()
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()
	userStore := db.Users()

	var createdUsers []*types.User
	for i, user := range users {
		newUser, err := userStore.Create(ctx, user)
		if err != nil {
			t.Errorf("could not create new user %s", err)
		}
		createdUsers = append(createdUsers, newUser)
		if i == 0 {
			createdUsers[i].SiteAdmin = true
		}
	}

	got, err := getUserCount(ctx, db)
	if err != nil {
		t.Errorf("could not get user count: %s", err)
	}

	want := 4
	if got != want {
		t.Errorf("got %d want %d", got, want)
	}
}

func TestGetSiteAdminEmails(t *testing.T) {
	logger := log.NoOp()
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()
	userStore := db.Users()

	var createdUsers []*types.User
	for i, user := range users {
		newUser, err := userStore.Create(ctx, user)
		if err != nil {
			t.Errorf("could not create new user %s", err)
		}
		createdUsers = append(createdUsers, newUser)

		if i == 0 || i == 2 {
			userStore.SetIsSiteAdmin(ctx, createdUsers[i].ID, true)
		}
	}

	got, _ := getSiteAdminEmails(ctx, db)
	want := []string{"test@test.com", "test3@test.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestGetUserEmail(t *testing.T) {
	logger := log.NoOp()
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()
	userStore := db.Users()

	cases := []struct {
		want string
		user database.NewUser
	}{
		{
			want: "test@test.com",
			user: users[0],
		},
		{
			want: "test2@test.com",
			user: users[1],
		},
		{
			want: "test3@test.com",
			user: users[2],
		},
		{
			want: "test4@test.com",
			user: users[3],
		},
	}

	for _, tc := range cases {
		newUser, err := userStore.Create(ctx, tc.user)
		if err != nil {
			t.Errorf("could not create new user: %s", err)
		}

		got, _, err := getUserEmail(ctx, db, newUser)
		if err != nil {
			t.Errorf("got an unexpected error: %s", err)
		}

		if got != tc.want {
			t.Errorf("got %s want %s", got, tc.want)
		}
	}
}

var users = []database.NewUser{
	{
		Email:                 "test@test.com",
		Username:              "test",
		DisplayName:           "test",
		Password:              "password",
		AvatarURL:             "avatar.jpg",
		EmailVerificationCode: "51235",
		EmailIsVerified:       false,
		FailIfNotInitialUser:  false,
		EnforcePasswordLength: false,
		TosAccepted:           false,
	},
	{
		Email:                 "test2@test.com",
		Username:              "test2",
		DisplayName:           "test2",
		Password:              "password",
		AvatarURL:             "avatar.jpg",
		EmailVerificationCode: "51235",
		EmailIsVerified:       false,
		FailIfNotInitialUser:  false,
		EnforcePasswordLength: false,
		TosAccepted:           false,
	},
	{
		Email:                 "test3@test.com",
		Username:              "test3",
		DisplayName:           "test3",
		Password:              "password",
		AvatarURL:             "avatar.jpg",
		EmailVerificationCode: "51235",
		EmailIsVerified:       false,
		FailIfNotInitialUser:  false,
		EnforcePasswordLength: false,
		TosAccepted:           false,
	},
	{
		Email:                 "test4@test.com",
		Username:              "test4",
		DisplayName:           "test4",
		Password:              "password",
		AvatarURL:             "avatar.jpg",
		EmailVerificationCode: "51235",
		EmailIsVerified:       false,
		FailIfNotInitialUser:  false,
		EnforcePasswordLength: false,
		TosAccepted:           false,
	},
}

var licensesToCreate = []struct {
	licenseId   string
	version     int
	licenseInfo license.Info
}{
	{
		licenseId: "b40537b3-d056-4235-afc2-1811cf9fa76e",
		version:   5,
		licenseInfo: license.Info{
			Tags:      []string{},
			UserCount: 10,
			ExpiresAt: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	},
	{
		licenseId: "9bbb0f96-b6c4-4545-926b-dd62ed3a7899",
		version:   12,
		licenseInfo: license.Info{
			Tags:      []string{},
			UserCount: 5,
			ExpiresAt: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	},
	{
		licenseId: "a8b0e28a-fc13-4724-b40a-12321202428b",
		version:   8,
		licenseInfo: license.Info{
			Tags:      []string{},
			UserCount: 30,
			ExpiresAt: time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	},
	{
		licenseId: "35e282b8-33c0-4eda-8225-8903a80e194f",
		version:   1,
		licenseInfo: license.Info{
			Tags:      []string{},
			UserCount: 27,
			ExpiresAt: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	},
}
