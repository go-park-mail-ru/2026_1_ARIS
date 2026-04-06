package friend

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/dto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type friendshipStorage struct {
	db *pgxpool.Pool
	// logger
}

type FriendshipRepo interface {
	GetFriends(ctx context.Context, profileID int64, status models.FriendshipStatus) ([]dto.FriendDTO, error)
	GetFriendshipStatus(ctx context.Context, profileID, friendID int64) (string, error)
	DeleteFriend(ctx context.Context, profileID, friendID int64) error
	GetFriendshipStatusBy(ctx context.Context, profileID, friendID int64) (string, error)
	GetOutgoingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error)
	GetIncomingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error)
	Create(ctx context.Context, profileID, friendID int64, status string) error
	AcceptFriendship(ctx context.Context, profileID1, profileID2 int64) error
	DeclineFriendship(ctx context.Context, profileID1, profileID2 int64) error
	RevokeFriendRequest(ctx context.Context, profileID, friendID int64) error
}

func NewFriendshipStorage(db *pgxpool.Pool) FriendshipRepo {
	return &friendshipStorage{
		db: db,
	}
}

func (storage *friendshipStorage) GetFriends(ctx context.Context, profileID int64, status models.FriendshipStatus) ([]dto.FriendDTO, error) {
	query := `
select p.avatar_id, p.id, up.first_name, up.last_name, ua.username, m.link, status, f.created_at, f.updated_at from 
	(select f.created_at, f.updated_at, status, case 
		when requester_id = $1 then addressee_id
		when addressee_id = $1 then requester_id
		end as friend
	from friendship f
	where $1 in (requester_id, addressee_id) AND status=$2) as f
join profile p on p.id=friend
join user_profile up on up.profile_id=friend
join user_account ua on up.user_account_id=ua.id
left join media m on p.avatar_id=m.id where m.mime_type='image' or mime_type is NULL ORDER BY p.id ASC;`

	rows, err := storage.db.Query(ctx, query, profileID, string(status))
	if err != nil {
		if pgxscan.NotFound(err) {
			return []dto.FriendDTO{}, nil
		}
		return []dto.FriendDTO{}, err
	}
	defer rows.Close()

	friends, err := pgx.CollectRows(rows, pgx.RowToStructByName[dto.FriendDTO])
	if err != nil {
		return nil, err
	}

	return friends, nil
}

func (storage *friendshipStorage) GetFriendshipStatus(ctx context.Context, profileID1, profileID2 int64) (string, error) {
	query := `
select status from friendship 
	where 
		least(requester_id, addressee_id)=least($1::bigint, $2::bigint) and 
		greatest(requester_id, addressee_id)=greatest($1::bigint, $2::bigint);`

	var status string

	row := storage.db.QueryRow(ctx, query, profileID1, profileID2)

	err := row.Scan(&status)
	if err != nil {
		if pgxscan.NotFound(err) {
			return "", xerrors.FriendshipNotFound
		}
		return "", err
	}

	return status, nil
}

func (storage *friendshipStorage) GetFriendshipStatusBy(ctx context.Context, profileID, friendID int64) (string, error) {
	query := `select status from friendship where requester_id=$1 and addressee_id=$2;`

	var status string

	row := storage.db.QueryRow(ctx, query, profileID, friendID)

	if err := row.Scan(&status); err != nil {
		if pgxscan.NotFound(err) {
			return "", nil
		}
		return "", err
	}

	return status, nil
}

func (storage *friendshipStorage) DeleteFriend(ctx context.Context, profileID, friendID int64) error {
	query := `
	DELETE FROM friendship 
		WHERE LEAST(requester_id, addressee_id)=LEAST($1::bigint, $2::bigint) AND 
		GREATEST(requester_id, addressee_id)=GREATEST($1::bigint, $2::bigint) AND 
		status='accepted'::friendship_status`

	res, err := storage.db.Exec(ctx, query, profileID, friendID)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return xerrors.NoRowsAffected
	}

	if res.RowsAffected() != 1 {
		return xerrors.MultipleRowsAffect
	}

	return nil
}

func (storage *friendshipStorage) GetOutgoingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error) {
	query := `
select f.addressee_id as id, p.avatar_id, up.first_name, up.last_name, ua.username, m.link, f.status, f.created_at, f.updated_at from friendship f
	join profile p on p.id=f.addressee_id
	join user_profile up on up.profile_id=p.id
	join user_account ua on ua.id=up.user_account_id 
	left join media m on m.id=p.avatar_id where f.requester_id=$1 AND f.status=$2;`

	rows, err := storage.db.Query(ctx, query, profileID, status)
	if err != nil {
		if pgxscan.NotFound(err) {
			return []dto.FriendDTO{}, nil
		}
		return []dto.FriendDTO{}, err
	}
	defer rows.Close()

	friends, err := pgx.CollectRows(rows, pgx.RowToStructByName[dto.FriendDTO])
	if err != nil {
		return nil, err
	}

	return friends, nil
}

func (storage *friendshipStorage) GetIncomingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error) {
	query := `
select f.requester_id as id, p.avatar_id, up.first_name, up.last_name, ua.username, m.link, f.status, f.created_at, f.updated_at from friendship f
	join profile p on p.id=f.requester_id
	join user_profile up on up.profile_id=p.id
	join user_account ua on ua.id=up.user_account_id 
	left join media m on m.id=p.avatar_id where f.addressee_id=$1 and f.status=$2;`

	rows, err := storage.db.Query(ctx, query, profileID, status)
	if err != nil {
		if pgxscan.NotFound(err) {
			return []dto.FriendDTO{}, nil
		}
		return []dto.FriendDTO{}, err
	}
	defer rows.Close()

	friends, err := pgx.CollectRows(rows, pgx.RowToStructByName[dto.FriendDTO])
	if err != nil {
		return nil, err
	}

	return friends, nil
}

func (storage *friendshipStorage) Create(ctx context.Context, profileID, friendID int64, status string) error {
	query := `insert into friendship (requester_id, addressee_id, status) values ($1, $2, $3)`

	res, err := storage.db.Exec(ctx, query, profileID, friendID, status)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return xerrors.NoRowsAffected
	}

	if res.RowsAffected() != 1 {
		return xerrors.MultipleRowsAffect
	}

	return nil
}

func (storage *friendshipStorage) AcceptFriendship(ctx context.Context, profileID1, profileID2 int64) error {
	query := `update friendship f set status='accepted' 
		where 
			least(requester_id, addressee_id)=least($1::bigint, $2::bigint) and
			greatest(requester_id, addressee_id)=greatest($1::bigint, $2::bigint) and
			status='pending'::friendship_status;`

	res, err := storage.db.Exec(ctx, query, profileID1, profileID2)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return xerrors.NoRowsAffected
	}

	if res.RowsAffected() > 1 {
		return xerrors.MultipleRowsAffect
	}

	return nil
}

func (storage *friendshipStorage) DeclineFriendship(ctx context.Context, profileID1, profileID2 int64) error {
	query := `DELETE FROM friendship WHERE LEAST(requester_id, addressee_id)=LEAST($1::bigint, $2::bigint) AND GREATEST(requester_id, addressee_id)=GREATEST($1::bigint, $2::bigint) AND status='pending'`

	res, err := storage.db.Exec(ctx, query, profileID1, profileID2)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return xerrors.NoRowsAffected
	}

	if res.RowsAffected() != 1 {
		return xerrors.MultipleRowsAffect
	}

	return nil
}

func (storage *friendshipStorage) RevokeFriendRequest(ctx context.Context, profileID, friendID int64) error {
	query := `DELETE FROM friendship where requester_id=$1 AND addressee_id=$2 AND status='pending'`

	res, err := storage.db.Exec(ctx, query, profileID, friendID)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return xerrors.NoRowsAffected
	}

	if res.RowsAffected() != 1 {
		return xerrors.MultipleRowsAffect
	}

	return nil
}
