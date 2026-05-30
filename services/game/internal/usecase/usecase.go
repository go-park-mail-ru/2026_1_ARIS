package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrForbidden         = errors.New("forbidden")
	ErrNotFound          = errors.New("not found")
	ErrRoomFull          = errors.New("room is full")
	ErrRoomTitleTaken    = errors.New("room title taken")
	ErrAlreadyStarted    = errors.New("game already started")
	ErrAlreadyAnswered   = errors.New("answer already submitted")
	ErrGamePaused        = errors.New("game paused")
	ErrPauseAlreadyUsed  = errors.New("pause already used")
	ErrActiveCreatedRoom = errors.New("active created room exists")
)

type Notifier func(context.Context, int64)

const (
	emptyWaitingRoomTTL    = 30 * time.Second
	staleWaitingMemberTTL  = 15 * time.Second
	gameStartCountdown     = 10 * time.Second
	defaultRoundPauseSec   = 5
	gamePauseDuration      = 2 * time.Minute
	forceResumeCountdown   = 5 * time.Second
	ratingBaseValue        = 1000
	ratingKFactor          = 32.0
	firstRatingSeasonYear  = 2026
	firstRatingSeasonMonth = time.May

	roundResultFirstCardDelay    = 260 * time.Millisecond
	roundResultNextCardBaseDelay = 1600 * time.Millisecond
	roundResultNextCardStepDelay = 1300 * time.Millisecond
	roundResultAnswerSettle      = 2100 * time.Millisecond
	roundResultTimesRevealGap    = 600 * time.Millisecond
	roundResultScoreStartGap     = 650 * time.Millisecond
	roundResultScoreStepDelay    = 1250 * time.Millisecond
	roundResultScoreboardSortGap = 650 * time.Millisecond
	roundResultTimerStartGap     = 850 * time.Millisecond
	publicLobbyMaxPlayers        = 80
	publicLobbyQuestionCount     = 5
	publicLobbyAnswerTimeoutSec  = 10
	publicLobbyRoundPauseSec     = 14
)

var (
	publicGuestNamePattern = regexp.MustCompile(`^[A-Za-zА-Яа-яЁё-]+$`)
)

type Service struct {
	store         repository.Store
	userClient    userpb.UserServiceClient
	supportClient supportpb.SupportServiceClient
	notify        Notifier
	timers        sync.Map
}

func New(store repository.Store, userClient userpb.UserServiceClient, supportClients ...supportpb.SupportServiceClient) *Service {
	var supportClient supportpb.SupportServiceClient
	if len(supportClients) > 0 {
		supportClient = supportClients[0]
	}
	return &Service{store: store, userClient: userClient, supportClient: supportClient}
}

func (s *Service) SetNotifier(notifier Notifier) {
	s.notify = notifier
}

func (s *Service) CreateRoom(ctx context.Context, userAccountID int64, in CreateRoomInput) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	if err := normalizeCreateInput(&in); err != nil {
		return Room{}, err
	}
	existingRoom, err := s.store.Rooms.GetWaitingCreatedByProfile(ctx, profileID)
	if err == nil {
		view, buildErr := s.buildRoom(ctx, *existingRoom, profileID)
		if buildErr != nil {
			return Room{}, buildErr
		}
		return view, ErrActiveCreatedRoom
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return Room{}, err
	}
	var room model.Room
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		var createErr error
		for i := 0; i < 5; i++ {
			room = model.Room{
				Uid:                uuid.New(),
				Title:              in.Title,
				InviteCode:         inviteCode(),
				GameType:           in.GameType,
				Status:             model.RoomStatusWaiting,
				CreatedByProfileID: profileID,
				MaxPlayers:         in.MaxPlayers,
				PasswordHash:       passwordHashPtr(in.Password),
				PasswordValue:      passwordValuePtr(in.Password),
				IsRanked:           in.IsRanked,
				QuestionCount:      in.QuestionCount,
				AnswerTimeoutSec:   in.AnswerTimeoutSec,
				RoundPauseSec:      in.RoundPauseSec,
			}
			createErr = tx.Rooms.Create(ctx, &room)
			if createErr == nil {
				break
			}
			if isRoomTitleUniqueViolation(createErr) {
				return ErrRoomTitleTaken
			}
			if !isUniqueViolation(createErr) {
				return createErr
			}
		}
		if createErr != nil {
			return createErr
		}
		return tx.Members.Add(ctx, room.ID, profileID)
	})
	if err != nil {
		return Room{}, err
	}
	return s.GetRoom(ctx, userAccountID, room.ID)
}

func (s *Service) CreatePublicRoom(ctx context.Context, userAccountID int64, in CreatePublicRoomInput) (Room, error) {
	if err := s.requireQuestionAdmin(ctx, userAccountID); err != nil {
		return Room{}, err
	}
	normalizePublicRoomInput(&in)
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	var room model.Room
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		var createErr error
		for i := 0; i < 5; i++ {
			code := inviteCode()
			room = model.Room{
				Uid:                uuid.New(),
				Title:              "Public lobby " + code,
				InviteCode:         code,
				GameType:           model.DefaultGameType,
				Status:             model.RoomStatusWaiting,
				CreatedByProfileID: profileID,
				MaxPlayers:         publicLobbyMaxPlayers,
				IsRanked:           false,
				IsPublicLobby:      true,
				QuestionCount:      publicLobbyQuestionCount,
				AnswerTimeoutSec:   in.AnswerTimeoutSec,
				RoundPauseSec:      in.RoundPauseSec,
			}
			createErr = tx.Rooms.Create(ctx, &room)
			if createErr == nil {
				break
			}
			if isRoomTitleUniqueViolation(createErr) {
				continue
			}
			if !isUniqueViolation(createErr) {
				return createErr
			}
		}
		return createErr
	})
	if err != nil {
		return Room{}, err
	}
	return s.GetRoom(ctx, userAccountID, room.ID)
}

func (s *Service) JoinPublicRoom(ctx context.Context, inviteCode string, firstName string, lastName string) (PublicJoinResult, error) {
	code := strings.ToUpper(strings.TrimSpace(inviteCode))
	if code == "" {
		return PublicJoinResult{}, ErrInvalidInput
	}
	firstName, lastName, err := normalizePublicGuestNames(firstName, lastName)
	if err != nil {
		return PublicJoinResult{}, err
	}
	token, err := publicGuestToken()
	if err != nil {
		return PublicJoinResult{}, err
	}
	tokenHash := publicGuestTokenHash(token)
	var (
		roomID    int64
		profileID int64
	)
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetByInviteCodeForUpdate(ctx, code)
		if err != nil {
			return mapRepoErr(err)
		}
		if !room.IsPublicLobby || room.Status != model.RoomStatusWaiting {
			return ErrForbidden
		}
		members, err := tx.Members.List(ctx, room.ID)
		if err != nil {
			return err
		}
		members = publicRoomPlayableMembers(*room, members)
		if len(members) >= room.MaxPlayers {
			return ErrRoomFull
		}
		profileID, err = tx.PublicParticipants.CreateGuestProfile(ctx)
		if err != nil {
			return err
		}
		participant := model.PublicParticipant{
			Uid:       uuid.New(),
			RoomID:    room.ID,
			ProfileID: profileID,
			TokenHash: tokenHash,
			FirstName: firstName,
			LastName:  lastName,
		}
		if err := tx.PublicParticipants.Create(ctx, &participant); err != nil {
			return err
		}
		if err := tx.Members.Add(ctx, room.ID, profileID); err != nil {
			return mapRepoErr(err)
		}
		if err := tx.Members.SetReady(ctx, room.ID, profileID, true); err != nil {
			return mapRepoErr(err)
		}
		roomID = room.ID
		return nil
	})
	if err != nil {
		return PublicJoinResult{}, err
	}
	s.notifyRoom(ctx, roomID)
	room, err := s.GetRoomByProfile(ctx, profileID, roomID)
	if err != nil {
		return PublicJoinResult{}, err
	}
	return PublicJoinResult{Token: token, Room: room}, nil
}

func (s *Service) JoinRoom(ctx context.Context, userAccountID int64, inviteCode string, roomIDRaw string, password string) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	if err := s.cleanupEmptyWaitingRooms(ctx); err != nil {
		return Room{}, err
	}
	code := strings.ToUpper(strings.TrimSpace(inviteCode))
	roomIDRaw = strings.TrimSpace(roomIDRaw)
	if code == "" && roomIDRaw == "" {
		return Room{}, ErrInvalidInput
	}
	var roomID int64
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		var (
			room *model.Room
			err  error
		)
		if roomIDRaw != "" {
			parsedRoomID, parseErr := strconv.ParseInt(roomIDRaw, 10, 64)
			if parseErr != nil || parsedRoomID <= 0 {
				return ErrInvalidInput
			}
			room, err = tx.Rooms.GetForUpdate(ctx, parsedRoomID)
		} else {
			room, err = tx.Rooms.GetByInviteCode(ctx, code)
		}
		if err != nil {
			return mapRepoErr(err)
		}
		roomID = room.ID
		if room.IsPublicLobby {
			if room.CreatedByProfileID == profileID {
				if room.Status == model.RoomStatusWaiting {
					if err := tx.Members.Deactivate(ctx, room.ID, profileID); err != nil && !errors.Is(mapRepoErr(err), ErrNotFound) {
						return mapRepoErr(err)
					}
				}
				return nil
			}
			members, err := tx.Members.List(ctx, room.ID)
			if err != nil {
				return err
			}
			members = publicRoomPlayableMembers(*room, members)
			for _, member := range members {
				if member.ProfileID == profileID {
					return nil
				}
			}
			if room.Status != model.RoomStatusWaiting {
				return ErrAlreadyStarted
			}
			if len(members) >= room.MaxPlayers {
				return ErrRoomFull
			}
			if err := tx.Members.Add(ctx, room.ID, profileID); err != nil {
				return mapRepoErr(err)
			}
			return mapRepoErr(tx.Members.SetReady(ctx, room.ID, profileID, true))
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		if room.CreatedByProfileID != profileID && !passwordMatches(room.PasswordHash, password) {
			return ErrForbidden
		}
		members, err := tx.Members.List(ctx, room.ID)
		if err != nil {
			return err
		}
		for _, member := range members {
			if member.ProfileID == profileID {
				return nil
			}
		}
		if len(members) >= room.MaxPlayers {
			return ErrRoomFull
		}
		if err := tx.Members.Add(ctx, room.ID, profileID); err != nil {
			return mapRepoErr(err)
		}
		return mapRepoErr(tx.Members.ClearReady(ctx, room.ID))
	})
	if err != nil {
		return Room{}, err
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoom(ctx, userAccountID, roomID)
}

func (s *Service) DisbandRoom(ctx context.Context, userAccountID, roomID int64) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.CreatedByProfileID != profileID {
			return ErrForbidden
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		return tx.Rooms.Deactivate(ctx, room.ID)
	})
	if err != nil {
		return err
	}
	s.notifyRoom(ctx, roomID)
	return nil
}

func (s *Service) LeaveRoom(ctx context.Context, userAccountID, roomID int64) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	return s.LeaveRoomByProfile(ctx, profileID, roomID)
}

func (s *Service) LeaveRoomByProfile(ctx context.Context, profileID, roomID int64) error {
	if profileID <= 0 || roomID <= 0 {
		return ErrInvalidInput
	}
	err := s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		if err := tx.Members.Deactivate(ctx, room.ID, profileID); err != nil {
			return mapRepoErr(err)
		}
		members, err := tx.Members.List(ctx, room.ID)
		if err != nil {
			return err
		}
		if len(members) == 0 {
			return tx.Rooms.TouchEmptyWaiting(ctx, room.ID)
		}
		return mapRepoErr(tx.Members.ClearReady(ctx, room.ID))
	})
	if err != nil {
		return err
	}
	s.notifyRoom(ctx, roomID)
	return nil
}

func (s *Service) LeaveWaitingRoomOnDisconnect(ctx context.Context, userAccountID, roomID int64) error {
	err := s.LeaveRoom(ctx, userAccountID, roomID)
	if errors.Is(err, ErrAlreadyStarted) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (s *Service) LeaveWaitingRoomOnDisconnectByProfile(ctx context.Context, profileID, roomID int64) error {
	err := s.LeaveRoomByProfile(ctx, profileID, roomID)
	if errors.Is(err, ErrAlreadyStarted) || errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (s *Service) TouchWaitingRoomMember(ctx context.Context, userAccountID, roomID int64) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	return s.TouchWaitingRoomMemberByProfile(ctx, profileID, roomID)
}

func (s *Service) TouchWaitingRoomMemberByProfile(ctx context.Context, profileID, roomID int64) error {
	if profileID <= 0 || roomID <= 0 {
		return ErrInvalidInput
	}
	return s.store.Members.TouchWaiting(ctx, roomID, profileID)
}

func (s *Service) KickPlayer(ctx context.Context, userAccountID, roomID, targetProfileID int64) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.CreatedByProfileID != profileID || targetProfileID == profileID {
			return ErrForbidden
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		return mapRepoErr(tx.Members.Deactivate(ctx, room.ID, targetProfileID))
	})
	if err != nil {
		return err
	}
	s.notifyRoom(ctx, roomID)
	return nil
}

func (s *Service) SetReady(ctx context.Context, userAccountID, roomID int64, isReady bool) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		return mapRepoErr(tx.Members.SetReady(ctx, room.ID, profileID, isReady))
	})
	if err != nil {
		return err
	}
	s.notifyRoom(ctx, roomID)
	return nil
}

func (s *Service) SetReplayReady(ctx context.Context, userAccountID, roomID int64, isReady bool) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	var deadline *time.Time
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusFinished {
			return ErrInvalidInput
		}
		if err := tx.Members.SetReady(ctx, room.ID, profileID, isReady); err != nil {
			return mapRepoErr(err)
		}
		members, err := tx.Members.List(ctx, room.ID)
		if err != nil {
			return err
		}
		if !areReplayMembersReady(members) {
			return nil
		}
		startAt, err := s.prepareReplay(ctx, tx, room)
		if err != nil {
			return err
		}
		deadline = &startAt
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	if deadline != nil {
		s.scheduleDeadline(roomID, *deadline)
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoom(ctx, userAccountID, roomID)
}

func (s *Service) UpdateRoomPassword(ctx context.Context, userAccountID, roomID int64, password string) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.CreatedByProfileID != profileID {
			return ErrForbidden
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		room.PasswordHash = passwordHashPtr(password)
		room.PasswordValue = passwordValuePtr(password)
		return tx.Rooms.Update(ctx, room)
	})
	if err != nil {
		return err
	}
	s.notifyRoom(ctx, roomID)
	return nil
}

func (s *Service) UpdateRoomTitle(ctx context.Context, userAccountID, roomID int64, title string) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	title, err = normalizeRoomTitle(title)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.CreatedByProfileID != profileID {
			return ErrForbidden
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		if room.Title == title {
			return nil
		}
		if err := tx.Rooms.UpdateTitle(ctx, room.ID, title); err != nil {
			if isRoomTitleUniqueViolation(err) {
				return ErrRoomTitleTaken
			}
			return mapRepoErr(err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.notifyRoom(ctx, roomID)
	return nil
}

func (s *Service) UpdateRoomRanked(ctx context.Context, userAccountID, roomID int64, isRanked bool) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.CreatedByProfileID != profileID {
			return ErrForbidden
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		room.IsRanked = isRanked
		if err := tx.Rooms.Update(ctx, room); err != nil {
			return mapRepoErr(err)
		}
		return mapRepoErr(tx.Members.ClearReady(ctx, room.ID))
	})
	if err != nil {
		return err
	}
	s.notifyRoom(ctx, roomID)
	return nil
}

func (s *Service) AssignAdmin(ctx context.Context, userAccountID, roomID, targetProfileID int64) error {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.CreatedByProfileID != profileID || targetProfileID == profileID {
			return ErrForbidden
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		ok, err := tx.Members.IsMember(ctx, room.ID, targetProfileID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		return mapRepoErr(tx.Rooms.UpdateAdmin(ctx, room.ID, targetProfileID))
	})
	if err != nil {
		return err
	}
	s.notifyRoom(ctx, roomID)
	return nil
}

func (s *Service) StartRoom(ctx context.Context, userAccountID, roomID int64) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	var deadline *time.Time
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.CreatedByProfileID != profileID {
			return ErrForbidden
		}
		if room.Status != model.RoomStatusWaiting {
			return ErrAlreadyStarted
		}
		members, err := tx.Members.List(ctx, room.ID)
		if err != nil {
			return err
		}
		members = publicRoomPlayableMembers(*room, members)
		if len(members) < 2 {
			return ErrInvalidInput
		}
		if !room.IsPublicLobby {
			for _, member := range members {
				if !member.IsReady {
					return ErrInvalidInput
				}
			}
		}
		questions, err := questionsForRoomStart(ctx, tx, room)
		if err != nil {
			return err
		}
		if len(questions) < room.QuestionCount {
			return ErrInvalidInput
		}
		for i, question := range questions[:room.QuestionCount] {
			if err := tx.RoomQs.Add(ctx, room.ID, question.ID, i+1); err != nil {
				return err
			}
		}
		startAt := time.Now().Add(gameStartCountdown)
		room.Status = model.RoomStatusActive
		room.CurrentQuestionIndex = 0
		room.CurrentQuestionID = nil
		room.QuestionStartedAt = nil
		room.QuestionDeadlineAt = nil
		room.NextQuestionAt = &startAt
		room.PausedByProfileID = nil
		room.PauseStartedAt = nil
		room.PauseUntilAt = nil
		if err := tx.Rooms.Update(ctx, room); err != nil {
			return err
		}
		if err := tx.Members.ClearReady(ctx, room.ID); err != nil {
			return err
		}
		deadline = &startAt
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	if deadline != nil {
		s.scheduleDeadline(roomID, *deadline)
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoom(ctx, userAccountID, roomID)
}

func areReplayMembersReady(members []model.RoomMember) bool {
	if len(members) < 2 {
		return false
	}
	for _, member := range members {
		if !member.IsReady {
			return false
		}
	}
	return true
}

func questionsForRoomStart(ctx context.Context, tx repository.Store, room *model.Room) ([]model.Question, error) {
	if room != nil && room.IsPublicLobby {
		return tx.Questions.PublicLobby(ctx)
	}
	return tx.Questions.Random(ctx, room.GameType, room.QuestionCount)
}

func (s *Service) prepareReplay(ctx context.Context, tx repository.Store, room *model.Room) (time.Time, error) {
	questions, err := questionsForRoomStart(ctx, tx, room)
	if err != nil {
		return time.Time{}, err
	}
	if len(questions) < room.QuestionCount {
		return time.Time{}, ErrInvalidInput
	}
	if err := tx.RoomQs.Clear(ctx, room.ID); err != nil {
		return time.Time{}, err
	}
	for i, question := range questions[:room.QuestionCount] {
		if err := tx.RoomQs.Add(ctx, room.ID, question.ID, i+1); err != nil {
			return time.Time{}, err
		}
	}
	if err := tx.Members.ResetForReplay(ctx, room.ID); err != nil {
		return time.Time{}, err
	}
	startAt := time.Now().Add(gameStartCountdown)
	room.Status = model.RoomStatusActive
	room.WinnerProfileID = nil
	room.CurrentQuestionIndex = 0
	room.CurrentQuestionID = nil
	room.QuestionStartedAt = nil
	room.QuestionDeadlineAt = nil
	room.NextQuestionAt = &startAt
	room.PausedByProfileID = nil
	room.PauseStartedAt = nil
	room.PauseUntilAt = nil
	room.FinishedAt = nil
	if err := tx.Rooms.Update(ctx, room); err != nil {
		return time.Time{}, err
	}
	return startAt, nil
}

func (s *Service) SubmitAnswer(ctx context.Context, userAccountID, roomID int64, value float64) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	return s.SubmitAnswerByProfile(ctx, profileID, roomID, value)
}

func (s *Service) SubmitPublicAnswer(ctx context.Context, roomID int64, token string, value float64) (Room, error) {
	participant, err := s.publicParticipantByToken(ctx, roomID, token)
	if err != nil {
		return Room{}, err
	}
	return s.SubmitAnswerByProfile(ctx, participant.ProfileID, roomID, value)
}

func (s *Service) SubmitAnswerByProfile(ctx context.Context, profileID, roomID int64, value float64) (Room, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || profileID <= 0 {
		return Room{}, ErrInvalidInput
	}
	var deadline *time.Time
	err := s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusActive {
			return ErrInvalidInput
		}
		now := time.Now()
		if isRoomPaused(room) {
			if room.PauseUntilAt == nil || now.Before(*room.PauseUntilAt) {
				return ErrGamePaused
			}
			next, err := resumePausedRoom(ctx, tx, room, *room.PauseUntilAt)
			if err != nil {
				return err
			}
			deadline = next
			now = time.Now()
		}
		if room.NextQuestionAt != nil {
			if now.Before(*room.NextQuestionAt) {
				deadline = room.NextQuestionAt
				return ErrInvalidInput
			}
			next, err := startQuestion(ctx, tx, room, room.CurrentQuestionIndex+1)
			if err != nil {
				return err
			}
			deadline = next
		}
		if room.QuestionDeadlineAt != nil && now.After(*room.QuestionDeadlineAt) {
			next, err := s.completeActiveQuestion(ctx, tx, room)
			if err != nil {
				return err
			}
			deadline = next
			return nil
		}
		ok, err := tx.Members.IsMember(ctx, roomID, profileID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		active, err := tx.RoomQs.GetActive(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		question, err := tx.Questions.Get(ctx, active.QuestionID)
		if err != nil {
			return mapRepoErr(err)
		}
		answer := model.Answer{
			RoomQuestionID: active.ID,
			ProfileID:      profileID,
			Answer:         value,
			Distance:       math.Abs(value - question.CorrectAnswer),
		}
		if err := tx.Answers.Add(ctx, &answer); err != nil {
			if isUniqueViolation(err) {
				return ErrAlreadyAnswered
			}
			return err
		}
		answerCount, err := tx.Answers.Count(ctx, active.ID)
		if err != nil {
			return err
		}
		members, err := tx.Members.List(ctx, roomID)
		if err != nil {
			return err
		}
		members = publicRoomPlayableMembers(*room, members)
		if answerCount >= len(members) {
			next, err := s.completeActiveQuestion(ctx, tx, room)
			if err != nil {
				return err
			}
			deadline = next
		}
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	if deadline != nil {
		s.scheduleDeadline(roomID, *deadline)
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoomByProfile(ctx, profileID, roomID)
}

func (s *Service) PauseRoom(ctx context.Context, userAccountID, roomID int64) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	var deadline *time.Time
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusActive {
			return ErrInvalidInput
		}
		now := time.Now()
		if isRoomPaused(room) {
			if room.PauseUntilAt != nil && !now.Before(*room.PauseUntilAt) {
				next, err := resumePausedRoom(ctx, tx, room, *room.PauseUntilAt)
				if err != nil {
					return err
				}
				deadline = next
			} else {
				return ErrGamePaused
			}
		}
		members, err := tx.Members.List(ctx, roomID)
		if err != nil {
			return err
		}
		var current *model.RoomMember
		for i := range members {
			if members[i].ProfileID == profileID {
				current = &members[i]
				break
			}
		}
		if current == nil {
			return ErrForbidden
		}
		if current.PauseUsed {
			return ErrPauseAlreadyUsed
		}
		pauseUntil := now.Add(gamePauseDuration)
		room.PausedByProfileID = &profileID
		room.PauseStartedAt = &now
		room.PauseUntilAt = &pauseUntil
		if err := tx.Members.SetPauseUsed(ctx, roomID, profileID); err != nil {
			return err
		}
		if err := tx.Members.ClearForceResumeRequests(ctx, roomID); err != nil {
			return err
		}
		if err := tx.Rooms.Update(ctx, room); err != nil {
			return err
		}
		deadline = &pauseUntil
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	if deadline != nil {
		s.scheduleDeadline(roomID, *deadline)
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoom(ctx, userAccountID, roomID)
}

func (s *Service) ForceResumeRoom(ctx context.Context, userAccountID, roomID int64) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	var deadline *time.Time
	err = s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusActive || !isRoomPaused(room) {
			return ErrInvalidInput
		}
		now := time.Now()
		if room.PauseUntilAt != nil && !now.Before(*room.PauseUntilAt) {
			next, err := resumePausedRoom(ctx, tx, room, *room.PauseUntilAt)
			if err != nil {
				return err
			}
			deadline = next
			return nil
		}
		if room.PausedByProfileID == nil || *room.PausedByProfileID == profileID {
			return ErrForbidden
		}
		ok, err := tx.Members.IsMember(ctx, roomID, profileID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.Members.SetForceResumeRequested(ctx, roomID, profileID, true); err != nil {
			return err
		}
		members, err := tx.Members.List(ctx, roomID)
		if err != nil {
			return err
		}
		votes, required := forceResumeVotes(members, *room.PausedByProfileID)
		if required > 0 && votes >= required {
			resumeAt := now.Add(forceResumeCountdown)
			if room.PauseUntilAt == nil || resumeAt.Before(*room.PauseUntilAt) {
				room.PauseUntilAt = &resumeAt
				if err := tx.Rooms.Update(ctx, room); err != nil {
					return err
				}
			}
			deadline = room.PauseUntilAt
		} else {
			deadline = room.PauseUntilAt
		}
		return nil
	})
	if err != nil {
		return Room{}, err
	}
	if deadline != nil {
		s.scheduleDeadline(roomID, *deadline)
	}
	s.notifyRoom(ctx, roomID)
	return s.GetRoom(ctx, userAccountID, roomID)
}

func (s *Service) FinalizeExpired(ctx context.Context, roomID int64) error {
	var nextDeadline *time.Time
	changed := false
	err := s.store.InTx(ctx, func(tx repository.Store) error {
		room, err := tx.Rooms.GetForUpdate(ctx, roomID)
		if err != nil {
			return mapRepoErr(err)
		}
		if room.Status != model.RoomStatusActive {
			return nil
		}
		now := time.Now()
		if isRoomPaused(room) {
			if room.PauseUntilAt == nil || now.Before(*room.PauseUntilAt) {
				nextDeadline = room.PauseUntilAt
				return nil
			}
			next, err := resumePausedRoom(ctx, tx, room, *room.PauseUntilAt)
			if err != nil {
				return err
			}
			nextDeadline = next
			changed = true
			now = time.Now()
		}
		if room.Status != model.RoomStatusActive {
			return nil
		}
		if room.NextQuestionAt != nil {
			if now.Before(*room.NextQuestionAt) {
				nextDeadline = room.NextQuestionAt
				return nil
			}
			next, err := startQuestion(ctx, tx, room, room.CurrentQuestionIndex+1)
			if err != nil {
				return err
			}
			nextDeadline = next
			changed = true
			return nil
		}
		if room.QuestionDeadlineAt == nil || now.Before(*room.QuestionDeadlineAt) {
			nextDeadline = room.QuestionDeadlineAt
			return nil
		}
		next, err := s.completeActiveQuestion(ctx, tx, room)
		if err != nil {
			return err
		}
		nextDeadline = next
		changed = true
		return nil
	})
	if err != nil {
		return err
	}
	if nextDeadline != nil {
		s.scheduleDeadline(roomID, *nextDeadline)
	}
	if changed {
		s.notifyRoom(ctx, roomID)
	}
	return nil
}

func (s *Service) GetRoom(ctx context.Context, userAccountID, roomID int64) (Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Room{}, err
	}
	return s.GetRoomByProfile(ctx, profileID, roomID)
}

func (s *Service) GetPublicRoom(ctx context.Context, roomID int64, token string) (Room, error) {
	participant, err := s.publicParticipantByToken(ctx, roomID, token)
	if err != nil {
		return Room{}, err
	}
	return s.GetRoomByProfile(ctx, participant.ProfileID, roomID)
}

func (s *Service) PublicProfileByToken(ctx context.Context, roomID int64, token string) (int64, error) {
	participant, err := s.publicParticipantByToken(ctx, roomID, token)
	if err != nil {
		return 0, err
	}
	return participant.ProfileID, nil
}

func (s *Service) GetRoomByProfile(ctx context.Context, profileID, roomID int64) (Room, error) {
	if profileID <= 0 || roomID <= 0 {
		return Room{}, ErrInvalidInput
	}
	if err := s.cleanupEmptyWaitingRooms(ctx); err != nil {
		return Room{}, err
	}
	room, err := s.store.Rooms.Get(ctx, roomID)
	if err != nil {
		return Room{}, mapRepoErr(err)
	}
	ok, err := s.store.Members.IsMember(ctx, roomID, profileID)
	if err != nil {
		return Room{}, err
	}
	if !ok && room.CreatedByProfileID != profileID {
		return Room{}, ErrForbidden
	}
	if room.Status == model.RoomStatusActive && roomEventDue(room, time.Now()) {
		_ = s.FinalizeExpired(ctx, roomID)
		room, err = s.store.Rooms.Get(ctx, roomID)
		if err != nil {
			return Room{}, mapRepoErr(err)
		}
	}
	return s.buildRoom(ctx, *room, profileID)
}

func (s *Service) ListRooms(ctx context.Context, userAccountID int64, limit, offset int) ([]Room, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if err := s.finalizeExpiredForProfile(ctx, profileID); err != nil {
		return nil, err
	}
	leftRoomIDs, err := s.store.Members.DeactivateWaitingForProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if err := s.cleanupEmptyWaitingRooms(ctx); err != nil {
		return nil, err
	}
	rooms, err := s.store.Rooms.ListForProfile(ctx, profileID, normalizeLimit(limit, 50, 100), normalizeOffset(offset))
	if err != nil {
		return nil, err
	}
	result := make([]Room, 0, len(rooms))
	for _, room := range rooms {
		view, err := s.buildRoom(ctx, room, profileID)
		if err == nil {
			result = append(result, view)
		}
	}
	for _, roomID := range leftRoomIDs {
		s.notifyRoom(ctx, roomID)
	}
	return result, nil
}

func (s *Service) History(ctx context.Context, userAccountID int64, limit, offset int) ([]HistoryItem, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if err := s.finalizeExpiredForProfile(ctx, profileID); err != nil {
		return nil, err
	}
	if err := s.cleanupEmptyWaitingRooms(ctx); err != nil {
		return nil, err
	}
	rooms, err := s.store.Rooms.HistoryForProfile(ctx, profileID, normalizeLimit(limit, 50, 100), normalizeOffset(offset))
	if err != nil {
		return nil, err
	}
	result := make([]HistoryItem, 0, len(rooms))
	for _, item := range rooms {
		view, err := s.buildRoom(ctx, item.Room, profileID)
		if err == nil {
			result = append(result, HistoryItem{Room: view, MyScore: item.MyScore, OpponentScore: item.OpponentScore})
		}
	}
	return result, nil
}

func (s *Service) Stats(ctx context.Context, userAccountID int64) (Stats, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return Stats{}, err
	}
	if err := s.finalizeExpiredForProfile(ctx, profileID); err != nil {
		return Stats{}, err
	}
	if err := s.cleanupEmptyWaitingRooms(ctx); err != nil {
		return Stats{}, err
	}
	stats, err := s.store.Members.Stats(ctx, profileID)
	if err != nil {
		return Stats{}, err
	}
	return Stats{Played: stats.Played, Won: stats.Won, Lost: stats.Lost, Drawn: stats.Drawn}, nil
}

func (s *Service) Leaderboard(ctx context.Context, gameType string, limit, offset int) (Leaderboard, error) {
	gameType = normalizeGameType(gameType)
	season := currentRatingSeason(gameType, time.Now())
	if err := s.store.Ratings.EnsureSeason(ctx, &season); err != nil {
		return Leaderboard{}, err
	}
	items, err := s.store.Ratings.Leaderboard(ctx, season.ID, normalizeLimit(limit, 50, 100), normalizeOffset(offset))
	if err != nil {
		return Leaderboard{}, err
	}
	entries := make([]LeaderboardEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, LeaderboardEntry{
			Rank:        item.Rank,
			ProfileID:   item.ProfileID,
			Player:      s.player(ctx, model.RoomMember{ProfileID: item.ProfileID}, 0, false),
			Rating:      item.Rating,
			GamesPlayed: item.GamesPlayed,
			Wins:        item.Wins,
			Draws:       item.Draws,
		})
	}
	return Leaderboard{
		GameType: gameType,
		Season:   mapRatingSeason(season),
		Entries:  entries,
	}, nil
}

func (s *Service) ListRoomMessages(ctx context.Context, userAccountID, roomID int64, limit, offset int) ([]RoomMessage, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	return s.ListRoomMessagesByProfile(ctx, profileID, roomID, limit, offset)
}

func (s *Service) ListRoomMessagesByProfile(ctx context.Context, profileID, roomID int64, limit, offset int) ([]RoomMessage, error) {
	if profileID <= 0 || roomID <= 0 {
		return nil, ErrInvalidInput
	}
	if err := s.ensureRoomChatAccess(ctx, roomID, profileID); err != nil {
		return nil, err
	}
	messages, err := s.store.Messages.List(ctx, roomID, normalizeLimit(limit, 100, 300), normalizeOffset(offset))
	if err != nil {
		return nil, err
	}
	return s.mapRoomMessages(ctx, messages, profileID), nil
}

func (s *Service) SendRoomMessage(ctx context.Context, userAccountID, roomID int64, text string) (RoomMessage, error) {
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return RoomMessage{}, err
	}
	return s.SendRoomMessageByProfile(ctx, profileID, roomID, text)
}

func (s *Service) SendRoomMessageByProfile(ctx context.Context, profileID, roomID int64, text string) (RoomMessage, error) {
	if profileID <= 0 || roomID <= 0 {
		return RoomMessage{}, ErrInvalidInput
	}
	text, err := normalizeRoomMessageText(text)
	if err != nil {
		return RoomMessage{}, err
	}
	if err := s.ensureRoomChatAccess(ctx, roomID, profileID); err != nil {
		return RoomMessage{}, err
	}
	message := model.RoomMessage{
		Uid:       uuid.New(),
		RoomID:    roomID,
		ProfileID: profileID,
		Text:      text,
	}
	if err := s.store.Messages.Add(ctx, &message); err != nil {
		return RoomMessage{}, err
	}
	return s.mapRoomMessage(ctx, message, profileID), nil
}

func (s *Service) CleanupEmptyWaitingRooms(ctx context.Context) error {
	return s.cleanupEmptyWaitingRooms(ctx)
}

func (s *Service) cleanupEmptyWaitingRooms(ctx context.Context) error {
	staleRoomIDs, err := s.store.Members.DeactivateStaleWaiting(ctx, staleWaitingMemberTTL)
	if err != nil {
		return err
	}
	for _, roomID := range staleRoomIDs {
		if err := s.store.Rooms.TouchEmptyWaiting(ctx, roomID); err != nil && !errors.Is(mapRepoErr(err), ErrNotFound) {
			return err
		}
	}
	return s.store.Rooms.DeactivateEmptyWaitingOlderThan(ctx, emptyWaitingRoomTTL)
}

func (s *Service) finalizeExpiredForProfile(ctx context.Context, profileID int64) error {
	roomIDs, err := s.store.Rooms.ExpiredActiveIDsForProfile(ctx, profileID)
	if err != nil {
		return err
	}
	for _, roomID := range roomIDs {
		if err := s.FinalizeExpired(ctx, roomID); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

func (s *Service) ListQuestions(ctx context.Context, gameType string, includeInactive bool, limit, offset int) ([]Question, error) {
	gameType = normalizeGameType(gameType)
	items, err := s.store.Questions.List(ctx, gameType, includeInactive, normalizeLimit(limit, 100, 200), normalizeOffset(offset))
	if err != nil {
		return nil, err
	}
	return mapQuestions(items), nil
}

func (s *Service) CreateQuestion(ctx context.Context, userAccountID int64, in QuestionInput) (Question, error) {
	if err := s.requireQuestionAdmin(ctx, userAccountID); err != nil {
		return Question{}, err
	}
	q, err := mapQuestionInput(in)
	if err != nil {
		return Question{}, err
	}
	if err := s.store.Questions.Create(ctx, &q); err != nil {
		return Question{}, err
	}
	return mapQuestion(q), nil
}

func (s *Service) UpdateQuestion(ctx context.Context, userAccountID, id int64, in QuestionInput) (Question, error) {
	if err := s.requireQuestionAdmin(ctx, userAccountID); err != nil {
		return Question{}, err
	}
	if id <= 0 {
		return Question{}, ErrInvalidInput
	}
	q, err := mapQuestionInput(in)
	if err != nil {
		return Question{}, err
	}
	q.ID = id
	if err := s.store.Questions.Update(ctx, &q); err != nil {
		return Question{}, mapRepoErr(err)
	}
	saved, err := s.store.Questions.Get(ctx, id)
	if err != nil {
		return Question{}, mapRepoErr(err)
	}
	return mapQuestion(*saved), nil
}

func (s *Service) DeleteQuestion(ctx context.Context, userAccountID, id int64) error {
	if err := s.requireQuestionAdmin(ctx, userAccountID); err != nil {
		return err
	}
	if id <= 0 {
		return ErrInvalidInput
	}
	return mapRepoErr(s.store.Questions.Delete(ctx, id))
}

func (s *Service) requireQuestionAdmin(ctx context.Context, userAccountID int64) error {
	if s.supportClient == nil {
		return ErrForbidden
	}
	profileID, err := s.profileIDByAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	resp, err := s.supportClient.GetProfileRole(ctx, &supportpb.GetProfileRoleRequest{ProfileId: profileID})
	if err != nil {
		return ErrForbidden
	}
	if resp.GetRole() != "admin" {
		return ErrForbidden
	}
	return nil
}

func (s *Service) profileIDByAccount(ctx context.Context, userAccountID int64) (int64, error) {
	if userAccountID <= 0 {
		return 0, ErrInvalidInput
	}
	resp, err := s.userClient.GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID})
	if err != nil {
		return 0, mapGRPCErr(err)
	}
	return resp.GetProfileId(), nil
}

func (s *Service) ensureRoomMember(ctx context.Context, roomID, profileID int64) error {
	if roomID <= 0 || profileID <= 0 {
		return ErrInvalidInput
	}
	if _, err := s.store.Rooms.Get(ctx, roomID); err != nil {
		return mapRepoErr(err)
	}
	ok, err := s.store.Members.IsMember(ctx, roomID, profileID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ensureRoomChatAccess(ctx context.Context, roomID, profileID int64) error {
	if roomID <= 0 || profileID <= 0 {
		return ErrInvalidInput
	}
	room, err := s.store.Rooms.Get(ctx, roomID)
	if err != nil {
		return mapRepoErr(err)
	}
	if room.IsPublicLobby && room.CreatedByProfileID == profileID {
		return nil
	}
	ok, err := s.store.Members.IsMember(ctx, roomID, profileID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) buildRoom(ctx context.Context, room model.Room, meProfileID int64) (Room, error) {
	members, err := s.store.Members.List(ctx, room.ID)
	if err != nil {
		return Room{}, err
	}
	members = publicRoomPlayableMembers(room, members)
	stats, err := s.store.Members.Stats(ctx, meProfileID)
	if err != nil {
		return Room{}, err
	}
	answeredProfiles := make(map[int64]bool)
	if room.CurrentQuestionID != nil {
		if active, err := s.store.RoomQs.GetActive(ctx, room.ID); err == nil {
			if answers, err := s.store.Answers.List(ctx, active.ID); err == nil {
				for _, answer := range answers {
					answeredProfiles[answer.ProfileID] = true
				}
			}
		}
	}
	players := make([]Player, 0, len(members))
	for _, member := range members {
		players = append(players, s.player(ctx, member, meProfileID, answeredProfiles[member.ProfileID]))
	}
	roomQuestions, err := s.store.RoomQs.List(ctx, room.ID)
	if err != nil {
		return Room{}, err
	}
	questions := make([]RoomQuestion, 0, len(roomQuestions))
	var current *CurrentQuestion
	for _, rq := range roomQuestions {
		q, err := s.store.Questions.Get(ctx, rq.QuestionID)
		if err != nil {
			continue
		}
		answers := []Answer{}
		if rq.Status == model.QuestionStatusCompleted {
			storedAnswers, _ := s.store.Answers.List(ctx, rq.ID)
			answers = mapAnswers(storedAnswers, rq.StartedAt)
		}
		view := RoomQuestion{
			Position:        rq.Position,
			Status:          rq.Status,
			Question:        mapQuestionForRoom(*q, rq.Status == model.QuestionStatusCompleted),
			WinnerProfileID: rq.WinnerProfileID,
			StartedAt:       timePtrString(rq.StartedAt),
			DeadlineAt:      timePtrString(rq.DeadlineAt),
			CompletedAt:     timePtrString(rq.CompletedAt),
			Answers:         answers,
		}
		questions = append(questions, view)
		if room.CurrentQuestionID != nil && *room.CurrentQuestionID == q.ID && rq.Status == model.QuestionStatusActive {
			current = &CurrentQuestion{
				Position:   rq.Position,
				ID:         q.ID,
				Text:       LocalizedText{RU: q.TextRU, EN: q.TextEN},
				StartedAt:  timePtrString(rq.StartedAt),
				DeadlineAt: timePtrString(rq.DeadlineAt),
			}
			current.HasAnswered = answeredProfiles[meProfileID]
		}
	}
	pauseVotes, pauseVotesRequired := forceResumeVotes(members, int64PtrValue(room.PausedByProfileID))
	ratingChanges, err := s.ratingChangesForRoom(ctx, room)
	if err != nil {
		return Room{}, err
	}
	return Room{
		ID:                      room.ID,
		Title:                   room.Title,
		InviteCode:              room.InviteCode,
		GameType:                room.GameType,
		Status:                  room.Status,
		CreatedByProfileID:      room.CreatedByProfileID,
		WinnerProfileID:         room.WinnerProfileID,
		MaxPlayers:              room.MaxPlayers,
		HasPassword:             room.PasswordHash != nil && strings.TrimSpace(*room.PasswordHash) != "",
		Password:                roomPasswordForProfile(room, meProfileID),
		IsRanked:                room.IsRanked,
		IsPublicLobby:           room.IsPublicLobby,
		QuestionCount:           room.QuestionCount,
		AnswerTimeoutSec:        room.AnswerTimeoutSec,
		RoundPauseSec:           room.RoundPauseSec,
		Creator:                 s.player(ctx, model.RoomMember{ProfileID: room.CreatedByProfileID}, meProfileID, false),
		CurrentQuestionIndex:    room.CurrentQuestionIndex,
		NextQuestionAt:          timePtrString(room.NextQuestionAt),
		PausedByProfileID:       room.PausedByProfileID,
		PauseStartedAt:          timePtrString(room.PauseStartedAt),
		PauseUntilAt:            timePtrString(room.PauseUntilAt),
		PauseForceVotes:         pauseVotes,
		PauseForceVotesRequired: pauseVotesRequired,
		CurrentQuestion:         current,
		Players:                 players,
		Questions:               questions,
		RatingChanges:           ratingChanges,
		ProfileStats:            Stats{Played: stats.Played, Won: stats.Won, Lost: stats.Lost, Drawn: stats.Drawn},
		CreatedAt:               room.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:               room.UpdatedAt.Format(time.RFC3339Nano),
		FinishedAt:              timePtrString(room.FinishedAt),
	}, nil
}

func (s *Service) player(ctx context.Context, member model.RoomMember, meProfileID int64, hasAnswered bool) Player {
	player := Player{ProfileID: member.ProfileID, Score: member.Score, IsReady: member.IsReady, HasAnswered: hasAnswered, PauseUsed: member.PauseUsed, ForceResumeRequested: member.ForceResumeRequested, IsMe: member.ProfileID == meProfileID}
	summary, err := s.userClient.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: member.ProfileID})
	if err == nil {
		player.UserAccountID = summary.GetUserAccountId()
		player.Username = summary.GetUsername()
		player.FirstName = summary.GetFirstName()
		player.LastName = summary.GetLastName()
		player.Name = strings.TrimSpace(summary.GetFirstName() + " " + summary.GetLastName())
		if player.Name == "" {
			player.Name = summary.GetUsername()
		}
		player.AvatarID = summary.AvatarId
	}
	if strings.TrimSpace(player.Name) == "" && s.store.PublicParticipants != nil {
		participant, err := s.store.PublicParticipants.GetByProfile(ctx, member.ProfileID)
		if err == nil {
			player.FirstName = participant.FirstName
			player.LastName = participant.LastName
			player.Name = strings.TrimSpace(participant.FirstName + " " + participant.LastName)
			player.Username = "guest"
		}
	}
	return player
}

func startQuestion(ctx context.Context, tx repository.Store, room *model.Room, position int) (*time.Time, error) {
	rq, err := tx.RoomQs.GetByPosition(ctx, room.ID, position)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	now := time.Now()
	deadline := now.Add(time.Duration(room.AnswerTimeoutSec) * time.Second)
	rq.Status = model.QuestionStatusActive
	rq.StartedAt = &now
	rq.DeadlineAt = &deadline
	if err := tx.RoomQs.Update(ctx, rq); err != nil {
		return nil, err
	}
	room.Status = model.RoomStatusActive
	room.CurrentQuestionIndex = position
	room.CurrentQuestionID = &rq.QuestionID
	room.QuestionStartedAt = &now
	room.QuestionDeadlineAt = &deadline
	room.NextQuestionAt = nil
	room.FinishedAt = nil
	if err := tx.Rooms.Update(ctx, room); err != nil {
		return nil, err
	}
	return &deadline, nil
}

func (s *Service) completeActiveQuestion(ctx context.Context, tx repository.Store, room *model.Room) (*time.Time, error) {
	active, err := tx.RoomQs.GetActive(ctx, room.ID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	answers, err := tx.Answers.List(ctx, active.ID)
	if err != nil {
		return nil, err
	}
	winner := answerWinner(answers)
	now := time.Now()
	active.Status = model.QuestionStatusCompleted
	active.WinnerProfileID = winner
	active.CompletedAt = &now
	if err := tx.RoomQs.Update(ctx, active); err != nil {
		return nil, err
	}
	if winner != nil {
		if err := tx.Members.IncrementScore(ctx, room.ID, *winner); err != nil {
			return nil, err
		}
	}
	members, err := tx.Members.List(ctx, room.ID)
	if err != nil {
		return nil, err
	}
	members = publicRoomPlayableMembers(*room, members)
	if active.Position >= room.QuestionCount {
		roomWinner := scoreWinner(members)
		room.Status = model.RoomStatusFinished
		room.WinnerProfileID = roomWinner
		room.CurrentQuestionID = nil
		room.QuestionStartedAt = nil
		room.QuestionDeadlineAt = nil
		room.NextQuestionAt = nil
		room.PausedByProfileID = nil
		room.PauseStartedAt = nil
		room.PauseUntilAt = nil
		room.FinishedAt = &now
		if err := tx.Members.ClearReady(ctx, room.ID); err != nil {
			return nil, err
		}
		if err := tx.Rooms.Update(ctx, room); err != nil {
			return nil, err
		}
		if room.IsRanked {
			if err := s.processRatingMatch(ctx, tx, room, members, now); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	nextQuestionAt := now.Add(roundResultTransitionDuration(room.RoundPauseSec, members, answers, active.StartedAt))
	room.Status = model.RoomStatusActive
	room.CurrentQuestionIndex = active.Position
	room.CurrentQuestionID = nil
	room.QuestionStartedAt = nil
	room.QuestionDeadlineAt = nil
	room.NextQuestionAt = &nextQuestionAt
	room.FinishedAt = nil
	if err := tx.Rooms.Update(ctx, room); err != nil {
		return nil, err
	}
	return &nextQuestionAt, nil
}

func roundResultCardDelay(revealIndex int) time.Duration {
	if revealIndex <= 0 {
		return roundResultFirstCardDelay
	}
	return roundResultNextCardBaseDelay + time.Duration(revealIndex-1)*roundResultNextCardStepDelay
}

func roundResultMemberAnswers(members []model.RoomMember, answers []model.Answer) []model.Answer {
	memberIDs := make(map[int64]struct{}, len(members))
	for _, member := range members {
		memberIDs[member.ProfileID] = struct{}{}
	}
	result := make([]model.Answer, 0, len(answers))
	for _, answer := range answers {
		if _, ok := memberIDs[answer.ProfileID]; ok {
			result = append(result, answer)
		}
	}
	return result
}

func publicRoomPlayableMembers(room model.Room, members []model.RoomMember) []model.RoomMember {
	if !room.IsPublicLobby || room.CreatedByProfileID <= 0 {
		return members
	}
	filtered := make([]model.RoomMember, 0, len(members))
	for _, member := range members {
		if member.ProfileID == room.CreatedByProfileID {
			continue
		}
		filtered = append(filtered, member)
	}
	return filtered
}

func roundResultAnswerGroupKey(value float64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func roundResultMaxRevealIndex(memberCount int, answers []model.Answer) int {
	if memberCount <= 0 {
		return 0
	}
	answeredProfiles := make(map[int64]struct{}, len(answers))
	answerGroups := make(map[string]struct{}, len(answers))
	for _, answer := range answers {
		answeredProfiles[answer.ProfileID] = struct{}{}
		answerGroups[roundResultAnswerGroupKey(answer.Answer)] = struct{}{}
	}
	groupCount := len(answerGroups)
	if len(answeredProfiles) < memberCount {
		groupCount++
	}
	return groupCount
}

func roundResultResponseTimeMs(answer model.Answer, startedAt *time.Time) int64 {
	if startedAt == nil {
		return 0
	}
	responseTimeMs := answer.AnsweredAt.Sub(*startedAt).Milliseconds()
	if responseTimeMs < 0 {
		return 0
	}
	return responseTimeMs
}

func roundResultResponseTimeBucket(answer model.Answer, startedAt *time.Time) int64 {
	return int64(math.Round(float64(roundResultResponseTimeMs(answer, startedAt)) / 10))
}

func roundResultAnswersTied(left, right model.Answer, startedAt *time.Time) bool {
	return left.Distance == right.Distance &&
		roundResultResponseTimeBucket(left, startedAt) == roundResultResponseTimeBucket(right, startedAt)
}

func roundResultPositivePointCount(memberCount int, answers []model.Answer, startedAt *time.Time) int {
	if memberCount <= 0 || len(answers) == 0 {
		return 0
	}
	entries := append([]model.Answer(nil), answers...)
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Distance != entries[j].Distance {
			return entries[i].Distance < entries[j].Distance
		}
		leftTime := roundResultResponseTimeMs(entries[i], startedAt)
		rightTime := roundResultResponseTimeMs(entries[j], startedAt)
		if leftTime != rightTime {
			return leftTime < rightTime
		}
		return entries[i].ProfileID < entries[j].ProfileID
	})

	positiveCount := 0
	for index := 0; index < len(entries); {
		end := index + 1
		for end < len(entries) && roundResultAnswersTied(entries[index], entries[end], startedAt) {
			end++
		}
		sum := 0
		for position := index; position < end; position++ {
			points := memberCount - position - 1
			if points > 0 {
				sum += points
			}
		}
		if sum > 0 {
			positiveCount += end - index
		}
		index = end
	}
	return positiveCount
}

func roundResultTransitionDuration(roundPauseSec int, members []model.RoomMember, answers []model.Answer, startedAt *time.Time) time.Duration {
	_ = members
	_ = answers
	_ = startedAt
	return time.Duration(normalizeRoundPauseSec(roundPauseSec)) * time.Second
}

func answerWinner(answers []model.Answer) *int64 {
	if len(answers) == 0 {
		return nil
	}
	best := answers[0]
	for _, answer := range answers[1:] {
		if answer.Distance < best.Distance ||
			(answer.Distance == best.Distance && answer.AnsweredAt.Before(best.AnsweredAt)) {
			best = answer
		}
	}
	return &best.ProfileID
}

func scoreWinner(members []model.RoomMember) *int64 {
	if len(members) == 0 {
		return nil
	}
	best := members[0]
	tie := false
	for _, member := range members[1:] {
		switch {
		case member.Score > best.Score:
			best = member
			tie = false
		case member.Score == best.Score:
			tie = true
		}
	}
	if tie {
		return nil
	}
	return &best.ProfileID
}

func (s *Service) processRatingMatch(ctx context.Context, tx repository.Store, room *model.Room, members []model.RoomMember, playedAt time.Time) error {
	if room == nil || !room.IsRanked || len(members) < 2 {
		return nil
	}
	season := currentRatingSeason(room.GameType, playedAt)
	if err := tx.Ratings.EnsureSeason(ctx, &season); err != nil {
		return err
	}
	profileIDs := memberProfileIDs(members)
	ratings, err := tx.Ratings.EnsurePlayerRatings(ctx, season.ID, room.GameType, profileIDs)
	if err != nil {
		return err
	}
	ratingByProfile := make(map[int64]int, len(ratings))
	for _, rating := range ratings {
		ratingByProfile[rating.ProfileID] = rating.Rating
	}
	groupHash := ratingGroupHash(profileIDs)
	dayStart := time.Date(playedAt.Year(), playedAt.Month(), playedAt.Day(), 0, 0, 0, 0, playedAt.Location())
	alreadyPlayed, err := tx.Ratings.CountMatchesForGroup(ctx, room.GameType, groupHash, dayStart, dayStart.AddDate(0, 0, 1))
	if err != nil {
		return err
	}
	occurrence := alreadyPlayed + 1
	weight := ratingWeightForOccurrence(occurrence)
	match := model.RatingMatch{
		Uid:             uuid.New(),
		RoomID:          room.ID,
		SeasonID:        season.ID,
		GameType:        room.GameType,
		GroupHash:       groupHash,
		GroupOccurrence: occurrence,
		RatingWeight:    weight,
		PlayedAt:        playedAt,
	}
	if err := tx.Ratings.AddMatch(ctx, &match); err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return err
	}
	deltas := ratingDeltas(members, ratingByProfile, weight)
	places := ratingPlaces(members)
	scoreCounts := ratingScoreCounts(members)
	topScore := 0
	for i, member := range members {
		if i == 0 || member.Score > topScore {
			topScore = member.Score
		}
	}
	for _, member := range members {
		before, ok := ratingByProfile[member.ProfileID]
		if !ok {
			before = ratingBaseValue
		}
		delta := deltas[member.ProfileID]
		after := before + delta
		if after < 0 {
			after = 0
		}
		isWin := member.Score == topScore && scoreCounts[topScore] == 1
		isDraw := scoreCounts[member.Score] > 1
		player := model.RatingMatchPlayer{
			MatchID:      match.ID,
			ProfileID:    member.ProfileID,
			Score:        member.Score,
			Place:        places[member.ProfileID],
			BeforeRating: before,
			AfterRating:  after,
			RatingDelta:  delta,
		}
		if err := tx.Ratings.AddMatchPlayer(ctx, &player); err != nil {
			return err
		}
		if err := tx.Ratings.ApplyPlayerRatingChange(ctx, season.ID, member.ProfileID, delta, isWin, isDraw); err != nil {
			return err
		}
	}
	return nil
}

func ratingDeltas(members []model.RoomMember, ratings map[int64]int, weight float64) map[int64]int {
	result := make(map[int64]int, len(members))
	if len(members) < 2 || weight <= 0 {
		return result
	}
	allScoresEqual := true
	firstScore := members[0].Score
	for _, member := range members[1:] {
		if member.Score != firstScore {
			allScoresEqual = false
			break
		}
	}
	if allScoresEqual {
		return result
	}
	raw := make(map[int64]float64, len(members))
	pairK := ratingKFactor / float64(len(members)-1)
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			left := members[i]
			right := members[j]
			leftActual := 0.5
			if left.Score > right.Score {
				leftActual = 1
			} else if left.Score < right.Score {
				leftActual = 0
			}
			rightActual := 1 - leftActual
			leftRating, ok := ratings[left.ProfileID]
			if !ok {
				leftRating = ratingBaseValue
			}
			rightRating, ok := ratings[right.ProfileID]
			if !ok {
				rightRating = ratingBaseValue
			}
			leftExpected := ratingExpectedScore(leftRating, rightRating)
			rightExpected := 1 - leftExpected
			raw[left.ProfileID] += pairK * (leftActual - leftExpected)
			raw[right.ProfileID] += pairK * (rightActual - rightExpected)
		}
	}
	for _, member := range members {
		result[member.ProfileID] = int(math.Round(raw[member.ProfileID] * weight))
	}
	return result
}

func ratingExpectedScore(leftRating, rightRating int) float64 {
	return 1 / (1 + math.Pow(10, float64(rightRating-leftRating)/400))
}

func ratingPlaces(members []model.RoomMember) map[int64]int {
	places := make(map[int64]int, len(members))
	for _, member := range members {
		place := 1
		for _, other := range members {
			if other.Score > member.Score {
				place++
			}
		}
		places[member.ProfileID] = place
	}
	return places
}

func ratingScoreCounts(members []model.RoomMember) map[int]int {
	counts := make(map[int]int, len(members))
	for _, member := range members {
		counts[member.Score]++
	}
	return counts
}

func memberProfileIDs(members []model.RoomMember) []int64 {
	ids := make([]int64, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.ProfileID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func ratingGroupHash(profileIDs []int64) string {
	ids := append([]int64(nil), profileIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, ":")))
	return hex.EncodeToString(sum[:])
}

func ratingWeightForOccurrence(occurrence int) float64 {
	switch occurrence {
	case 1:
		return 1
	case 2:
		return 0.5
	case 3:
		return 0.25
	default:
		return 0
	}
}

func currentRatingSeason(gameType string, now time.Time) model.RatingSeason {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0)
	seasonNumber := (start.Year()-firstRatingSeasonYear)*12 + int(start.Month()-firstRatingSeasonMonth) + 1
	if seasonNumber < 1 {
		seasonNumber = 1
	}
	return model.RatingSeason{
		Uid:          uuid.New(),
		GameType:     gameType,
		SeasonNumber: seasonNumber,
		SeasonYear:   start.Year(),
		SeasonMonth:  int(start.Month()),
		Title:        fmt.Sprintf("Сезон %d: %s %d", seasonNumber, russianMonthName(start.Month()), start.Year()),
		StartsAt:     start,
		EndsAt:       end,
	}
}

func russianMonthName(month time.Month) string {
	names := map[time.Month]string{
		time.January:   "Январь",
		time.February:  "Февраль",
		time.March:     "Март",
		time.April:     "Апрель",
		time.May:       "Май",
		time.June:      "Июнь",
		time.July:      "Июль",
		time.August:    "Август",
		time.September: "Сентябрь",
		time.October:   "Октябрь",
		time.November:  "Ноябрь",
		time.December:  "Декабрь",
	}
	if name, ok := names[month]; ok {
		return name
	}
	return month.String()
}

func mapRatingSeason(season model.RatingSeason) RatingSeason {
	return RatingSeason{
		SeasonNumber: season.SeasonNumber,
		Title:        season.Title,
		StartsAt:     season.StartsAt.Format(time.RFC3339Nano),
		EndsAt:       season.EndsAt.Format(time.RFC3339Nano),
	}
}

func (s *Service) ratingChangesForRoom(ctx context.Context, room model.Room) ([]RatingChange, error) {
	if !room.IsRanked || room.Status != model.RoomStatusFinished {
		return nil, nil
	}
	items, err := s.store.Ratings.RatingChangesForRoom(ctx, room.ID)
	if err != nil {
		return nil, err
	}
	result := make([]RatingChange, 0, len(items))
	for _, item := range items {
		result = append(result, RatingChange{
			ProfileID:    item.ProfileID,
			Score:        item.Score,
			Place:        item.Place,
			BeforeRating: item.BeforeRating,
			AfterRating:  item.AfterRating,
			RatingDelta:  item.RatingDelta,
			RatingWeight: item.RatingWeight,
			SeasonNumber: item.SeasonNumber,
			SeasonTitle:  item.SeasonTitle,
		})
	}
	return result, nil
}

func isRoomPaused(room *model.Room) bool {
	return room != nil && room.PausedByProfileID != nil && room.PauseStartedAt != nil && room.PauseUntilAt != nil
}

func roomEventDue(room *model.Room, now time.Time) bool {
	if room == nil || room.Status != model.RoomStatusActive {
		return false
	}
	if isRoomPaused(room) {
		return room.PauseUntilAt != nil && !now.Before(*room.PauseUntilAt)
	}
	if room.NextQuestionAt != nil && !now.Before(*room.NextQuestionAt) {
		return true
	}
	return room.QuestionDeadlineAt != nil && !now.Before(*room.QuestionDeadlineAt)
}

func nextRoomEvent(room *model.Room) *time.Time {
	if room == nil || room.Status != model.RoomStatusActive {
		return nil
	}
	if isRoomPaused(room) {
		return room.PauseUntilAt
	}
	if room.NextQuestionAt != nil {
		return room.NextQuestionAt
	}
	return room.QuestionDeadlineAt
}

func resumePausedRoom(ctx context.Context, tx repository.Store, room *model.Room, resumeAt time.Time) (*time.Time, error) {
	if !isRoomPaused(room) {
		return nextRoomEvent(room), nil
	}
	elapsed := resumeAt.Sub(*room.PauseStartedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > gamePauseDuration {
		elapsed = gamePauseDuration
	}
	if room.QuestionDeadlineAt != nil {
		shifted := room.QuestionDeadlineAt.Add(elapsed)
		room.QuestionDeadlineAt = &shifted
		if active, err := tx.RoomQs.GetActive(ctx, room.ID); err == nil && active.DeadlineAt != nil {
			active.DeadlineAt = &shifted
			if err := tx.RoomQs.Update(ctx, active); err != nil {
				return nil, err
			}
		} else if err != nil && !errors.Is(mapRepoErr(err), ErrNotFound) {
			return nil, err
		}
	}
	if room.NextQuestionAt != nil {
		shifted := room.NextQuestionAt.Add(elapsed)
		room.NextQuestionAt = &shifted
	}
	room.PausedByProfileID = nil
	room.PauseStartedAt = nil
	room.PauseUntilAt = nil
	if err := tx.Members.ClearForceResumeRequests(ctx, room.ID); err != nil {
		return nil, err
	}
	if err := tx.Rooms.Update(ctx, room); err != nil {
		return nil, err
	}
	return nextRoomEvent(room), nil
}

func forceResumeVotes(members []model.RoomMember, pausedByProfileID int64) (int, int) {
	if pausedByProfileID <= 0 {
		return 0, 0
	}
	votes := 0
	required := 0
	for _, member := range members {
		if member.ProfileID == pausedByProfileID {
			continue
		}
		required++
		if member.ForceResumeRequested {
			votes++
		}
	}
	return votes, required
}

func (s *Service) scheduleDeadline(roomID int64, deadline time.Time) {
	key := fmt.Sprintf("%d:%d", roomID, deadline.UnixNano())
	if _, loaded := s.timers.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	go func() {
		defer s.timers.Delete(key)
		delay := time.Until(deadline)
		if delay > 0 {
			time.Sleep(delay + 150*time.Millisecond)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.FinalizeExpired(ctx, roomID)
	}()
}

func (s *Service) notifyRoom(ctx context.Context, roomID int64) {
	if s.notify != nil {
		s.notify(ctx, roomID)
	}
}

func mapQuestionInput(in QuestionInput) (model.Question, error) {
	gameType := normalizeGameType(in.GameType)
	textRU := strings.TrimSpace(in.Text.RU)
	textEN := strings.TrimSpace(in.Text.EN)
	if !validQuestionText(textRU) || !validQuestionText(textEN) || math.IsNaN(in.CorrectAnswer) || math.IsInf(in.CorrectAnswer, 0) {
		return model.Question{}, ErrInvalidInput
	}
	return model.Question{
		Uid:           uuid.New(),
		GameType:      gameType,
		TextRU:        textRU,
		TextEN:        textEN,
		CorrectAnswer: in.CorrectAnswer,
		IsActive:      in.IsActive,
	}, nil
}

func validQuestionText(text string) bool {
	length := len([]rune(text))
	return length >= 5 && length <= 512
}

func mapQuestions(items []model.Question) []Question {
	result := make([]Question, 0, len(items))
	for _, item := range items {
		result = append(result, mapQuestion(item))
	}
	return result
}

func mapQuestion(item model.Question) Question {
	return Question{
		ID:            item.ID,
		Text:          LocalizedText{RU: item.TextRU, EN: item.TextEN},
		CorrectAnswer: item.CorrectAnswer,
		IsActive:      item.IsActive,
	}
}

func mapQuestionForRoom(item model.Question, revealAnswer bool) Question {
	q := mapQuestion(item)
	if !revealAnswer {
		q.CorrectAnswer = 0
	}
	return q
}

func mapAnswers(items []model.Answer, startedAt *time.Time) []Answer {
	result := make([]Answer, 0, len(items))
	for _, item := range items {
		var responseTimeMs int64
		if startedAt != nil {
			responseTimeMs = item.AnsweredAt.Sub(*startedAt).Milliseconds()
			if responseTimeMs < 0 {
				responseTimeMs = 0
			}
		}
		result = append(result, Answer{
			ProfileID:      item.ProfileID,
			Answer:         item.Answer,
			Distance:       item.Distance,
			AnsweredAt:     item.AnsweredAt.Format(time.RFC3339Nano),
			ResponseTimeMs: responseTimeMs,
		})
	}
	return result
}

func (s *Service) mapRoomMessages(ctx context.Context, items []model.RoomMessage, meProfileID int64) []RoomMessage {
	result := make([]RoomMessage, 0, len(items))
	for _, item := range items {
		result = append(result, s.mapRoomMessage(ctx, item, meProfileID))
	}
	return result
}

func (s *Service) mapRoomMessage(ctx context.Context, item model.RoomMessage, meProfileID int64) RoomMessage {
	return RoomMessage{
		ID:        item.ID,
		RoomID:    item.RoomID,
		ProfileID: item.ProfileID,
		Text:      item.Text,
		Author:    s.player(ctx, model.RoomMember{ProfileID: item.ProfileID}, meProfileID, false),
		CreatedAt: item.CreatedAt.Format(time.RFC3339Nano),
	}
}

func normalizeCreateInput(in *CreateRoomInput) error {
	title, err := normalizeRoomTitle(in.Title)
	if err != nil {
		return err
	}
	in.Title = title
	in.GameType = normalizeGameType(in.GameType)
	if in.MaxPlayers < 2 {
		in.MaxPlayers = 2
	}
	if in.MaxPlayers > 8 {
		in.MaxPlayers = 8
	}
	if in.QuestionCount <= 0 || in.QuestionCount > 25 {
		in.QuestionCount = 5
	}
	if in.AnswerTimeoutSec <= 0 {
		in.AnswerTimeoutSec = 10
	}
	if in.AnswerTimeoutSec < 3 {
		in.AnswerTimeoutSec = 3
	}
	if in.AnswerTimeoutSec > 120 {
		in.AnswerTimeoutSec = 120
	}
	in.RoundPauseSec = normalizeRoundPauseSec(in.RoundPauseSec)
	return nil
}

func normalizePublicRoomInput(in *CreatePublicRoomInput) {
	if in.AnswerTimeoutSec <= 0 {
		in.AnswerTimeoutSec = publicLobbyAnswerTimeoutSec
	}
	if in.AnswerTimeoutSec < 3 {
		in.AnswerTimeoutSec = 3
	}
	if in.AnswerTimeoutSec > 120 {
		in.AnswerTimeoutSec = 120
	}
	if in.RoundPauseSec <= 0 {
		in.RoundPauseSec = publicLobbyRoundPauseSec
	} else {
		in.RoundPauseSec = normalizeRoundPauseSec(in.RoundPauseSec)
	}
}

func normalizeRoundPauseSec(value int) int {
	if value <= 0 {
		return defaultRoundPauseSec
	}
	if value < 1 {
		return 1
	}
	if value > 60 {
		return 60
	}
	return value
}

func normalizeRoomTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 30 {
		return "", ErrInvalidInput
	}
	return title, nil
}

func normalizeRoomMessageText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) > 500 {
		return "", ErrInvalidInput
	}
	return text, nil
}

func normalizePublicGuestNames(firstName string, lastName string) (string, string, error) {
	firstName, err := normalizePublicGuestName(firstName)
	if err != nil {
		return "", "", err
	}
	lastName, err = normalizePublicGuestName(lastName)
	if err != nil {
		return "", "", err
	}
	if detectPublicGuestAlphabet(firstName) != detectPublicGuestAlphabet(lastName) {
		return "", "", ErrInvalidInput
	}
	return firstName, lastName, nil
}

func normalizePublicGuestName(value string) (string, error) {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) == 0 || len(runes) > 12 || !publicGuestNamePattern.MatchString(value) {
		return "", ErrInvalidInput
	}
	hasLatin := false
	hasCyrillic := false
	for _, r := range runes {
		switch {
		case r == '-':
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			hasLatin = true
		case (r >= 'А' && r <= 'я') || r == 'Ё' || r == 'ё':
			hasCyrillic = true
		default:
			return "", ErrInvalidInput
		}
	}
	if hasLatin == hasCyrillic {
		return "", ErrInvalidInput
	}
	for i, r := range runes {
		if i == 0 {
			runes[i] = unicode.ToUpper(r)
		} else {
			runes[i] = unicode.ToLower(r)
		}
	}
	return string(runes), nil
}

func detectPublicGuestAlphabet(value string) string {
	for _, r := range value {
		switch {
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			return "latin"
		case (r >= 'А' && r <= 'я') || r == 'Ё' || r == 'ё':
			return "cyrillic"
		}
	}
	return ""
}

func publicGuestToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func publicGuestTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func (s *Service) publicParticipantByToken(ctx context.Context, roomID int64, token string) (*model.PublicParticipant, error) {
	if roomID <= 0 || strings.TrimSpace(token) == "" || s.store.PublicParticipants == nil {
		return nil, ErrInvalidInput
	}
	participant, err := s.store.PublicParticipants.GetByTokenHash(ctx, publicGuestTokenHash(token))
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if participant.RoomID != roomID {
		return nil, ErrForbidden
	}
	room, err := s.store.Rooms.Get(ctx, roomID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if !room.IsPublicLobby {
		return nil, ErrForbidden
	}
	return participant, nil
}

func passwordHashPtr(password string) *string {
	password = strings.TrimSpace(password)
	if password == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(password))
	hash := hex.EncodeToString(sum[:])
	return &hash
}

func passwordValuePtr(password string) *string {
	password = strings.TrimSpace(password)
	if password == "" {
		return nil
	}
	return &password
}

func roomPasswordForProfile(room model.Room, profileID int64) string {
	if room.CreatedByProfileID != profileID || room.PasswordValue == nil {
		return ""
	}
	return strings.TrimSpace(*room.PasswordValue)
}

func passwordMatches(hash *string, password string) bool {
	if hash == nil || strings.TrimSpace(*hash) == "" {
		return true
	}
	candidate := passwordHashPtr(password)
	return candidate != nil && *candidate == *hash
}

func normalizeGameType(gameType string) string {
	gameType = strings.TrimSpace(gameType)
	if gameType == "" {
		return model.DefaultGameType
	}
	return gameType
}

func normalizeLimit(value, fallback, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func normalizeOffset(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func timePtrString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	out := value.Format(time.RFC3339Nano)
	return &out
}

func int64PtrValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func inviteCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 6)
	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return strings.ToUpper(uuid.NewString()[:6])
	}
	for i, b := range randomBytes {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(buf)
}

func mapRepoErr(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func mapGRPCErr(err error) error {
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if status.Code(err) == codes.InvalidArgument {
		return ErrInvalidInput
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isRoomTitleUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "game_room_active_waiting_title_unique_idx"
}
