package dbrepo

import (
	"context"
	"errors"
	"time"

	"github.com/ashparshp/bookings/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (m *postgresDBRepo) AllUsers() bool {
	return true
}

// InsertReservation inserts a reservation into the database
func (m *postgresDBRepo) InsertReservation(res models.Reservation) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newID int
	stmt := `INSERT INTO reservations (first_name, last_name, email, phone, start_date, end_date, room_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) returning id`

	err := m.DB.QueryRowContext(ctx, stmt, res.FirstName, res.LastName, res.Email, res.Phone, res.StartDate, res.EndDate, res.RoomID, time.Now(), time.Now()).Scan(&newID)
	if err != nil {
		return 0, err
	}

	return newID, nil
}

// InsertRoomRestriction inserts a room restriction into the database
func (m *postgresDBRepo) InsertRoomRestriction(r models.RoomRestriction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO room_restrictions (start_date, end_date, room_id, reservation_id, restriction_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := m.DB.ExecContext(ctx, stmt, r.StartDate, r.EndDate, r.RoomID, r.ReservationID, r.RestrictionID, time.Now(), time.Now())
	if err != nil {
		return err
	}

	return nil
}

// SearchAvailabilityByDatesByRoomID returns true if there are available rooms for the given dates
func (m *postgresDBRepo) SearchAvailabilityByDatesByRoomID(start, end time.Time, roomID int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `select count(id) from room_restrictions where room_id = $1 and $2 < end_date and $3 > start_date`
	var numRows int
	row := m.DB.QueryRowContext(ctx, query, roomID, start, end)
	err := row.Scan(&numRows)
	if err != nil {
		return false, err
	}

	if numRows == 0 {
		return true, nil
	}
	return false, nil
}

// SearchAvailabilityForAllRooms returns a slice of available rooms for the given dates
func (m *postgresDBRepo) SearchAvailabilityForAllRooms(start, end time.Time) ([]models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var rooms []models.Room

	query := `select r.id, r.room_name from rooms r
	where r.id not in
	(select room_id from room_restrictions rr where $1 < rr.end_date and $2 > rr.start_date)`

	rows, err := m.DB.QueryContext(ctx, query, start, end)
	if err != nil {
		return rooms, err
	}
	defer rows.Close()

	for rows.Next() {
		var room models.Room
		err := rows.Scan(&room.ID, &room.RoomName)
		if err != nil {
			return rooms, err
		}
		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		return rooms, err
	}

	return rooms, nil
}

// GetRoomByID returns a room by its ID
func (m *postgresDBRepo) GetRoomByID(id int) (models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var room models.Room
	query := `select id, room_name, created_at, updated_at from rooms where id = $1`
	row := m.DB.QueryRowContext(ctx, query, id)
	err := row.Scan(&room.ID, &room.RoomName, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		return room, err
	}

	return room, nil
}

// GetUserByID returns a user by its ID
func (m *postgresDBRepo) GetUserByID(id int) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user models.User
	query := `select id, first_name, last_name, email, password, access_level, created_at, updated_at from users where id = $1`
	
	row := m.DB.QueryRowContext(ctx, query, id)
	
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password, &user.AccessLevel, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return user, err
	}
	return user, nil
}

// UpdateUser updates a user in the database
func (m *postgresDBRepo) UpdateUser(u models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE users SET first_name = $1, last_name = $2, email = $3, access_level = $4, updated_at = $5`

	_, err := m.DB.ExecContext(ctx, stmt, u.FirstName, u.LastName, u.Email, u.AccessLevel, time.Now())
	if err != nil {
		return err
	}

	return nil
}

// AuthenticateUser checks if the user exists and verifies the password
func (m *postgresDBRepo) AuthenticateUser(email, testPassword string) (int, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	query := `SELECT id, password FROM users WHERE email = $1`
	row := m.DB.QueryRowContext(ctx, query, email)

	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		return 0, "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(testPassword))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, "", errors.New("incorrect password")
	} else if err != nil {
		return 0, "", err
	}
	return id, hashedPassword, nil
}

// AllReservations returns all reservations from the database
func (m *postgresDBRepo) AllReservations() ([]models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservations []models.Reservation

	query := `
		SELECT 
			r.id, r.first_name, r.last_name, r.email, r.phone, 
			r.start_date, r.end_date, r.room_id, 
			r.created_at, r.updated_at, r.processed,
			rm.id, rm.room_name
		FROM reservations r
		LEFT JOIN rooms rm ON r.room_id = rm.id
		ORDER BY r.start_date ASC
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var res models.Reservation
		err := rows.Scan(
			&res.ID,
			&res.FirstName,
			&res.LastName,
			&res.Email,
			&res.Phone,
			&res.StartDate,
			&res.EndDate,
			&res.RoomID,
			&res.CreatedAt,
			&res.UpdatedAt,
			&res.Processed,
			&res.Room.ID,
			&res.Room.RoomName,
		)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, res)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return reservations, nil
}

// AllNewReservations returns all new reservations that have not been processed
func (m *postgresDBRepo) AllNewReservations() ([]models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservations []models.Reservation

	query := `
		SELECT 
			r.id, r.first_name, r.last_name, r.email, r.phone, 
			r.start_date, r.end_date, r.room_id, 
			r.created_at, r.updated_at,
			rm.id, rm.room_name
		FROM reservations r
		LEFT JOIN rooms rm ON r.room_id = rm.id
		WHERE processed = 0
		ORDER BY r.start_date ASC
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var res models.Reservation
		err := rows.Scan(
			&res.ID,
			&res.FirstName,
			&res.LastName,
			&res.Email,
			&res.Phone,
			&res.StartDate,
			&res.EndDate,
			&res.RoomID,
			&res.CreatedAt,
			&res.UpdatedAt,
			&res.Room.ID,
			&res.Room.RoomName,
		)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, res)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return reservations, nil
}

// GetReservationByID returns a reservation by its ID
func (m *postgresDBRepo) GetReservationByID(id int) (models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var res models.Reservation
	query := `
		SELECT 
			r.id, r.first_name, r.last_name, r.email, r.phone, 
			r.start_date, r.end_date, r.room_id, 
			r.created_at, r.updated_at, r.processed,
			rm.id, rm.room_name
		FROM reservations r
		LEFT JOIN rooms rm ON r.room_id = rm.id
		WHERE r.id = $1
	`

	row := m.DB.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&res.ID,
		&res.FirstName,
		&res.LastName,
		&res.Email,
		&res.Phone,
		&res.StartDate,
		&res.EndDate,
		&res.RoomID,
		&res.CreatedAt,
		&res.UpdatedAt,
		&res.Processed,
		&res.Room.ID,
		&res.Room.RoomName,
	)
	if err != nil {
		return res, err
	}

	return res, nil
}

// UpdateReservation updates a reservation in the database
func (m *postgresDBRepo) UpdateReservation(u models.Reservation, id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE reservations SET first_name = $1, last_name = $2, email = $3, phone = $4, updated_at = $5 WHERE id = $6`

	_, err := m.DB.ExecContext(ctx, stmt, u.FirstName, u.LastName, u.Email, u.Phone, time.Now(), id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteReservation deletes a reservation from the database
func (m *postgresDBRepo) DeleteReservation(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `DELETE FROM reservations WHERE id = $1`

	_, err := m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	return nil
}

// UpdateProcessedForReservation updates the processed status of a reservation
func (m *postgresDBRepo) UpdateProcessedForReservation(id, processed int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE reservations SET processed = $1 WHERE id = $2`

	_, err := m.DB.ExecContext(ctx, stmt, processed, id)
	if err != nil {
		return err
	}

	return nil
}

// AllRooms returns all rooms from the database
func (m *postgresDBRepo) AllRooms() ([]models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var rooms []models.Room

	query := `SELECT id, room_name, created_at, updated_at FROM rooms ORDER BY room_name`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var room models.Room
		err := rows.Scan(&room.ID, &room.RoomName, &room.CreatedAt, &room.UpdatedAt)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

// GetRestrictionsForRoomByDate returns room restrictions for a specific room and date
func (m *postgresDBRepo) GetRestrictionsForRoomByDate(roomID int, start, end time.Time) ([]models.RoomRestriction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var restrictions []models.RoomRestriction

	query := `SELECT id, start_date, end_date, room_id, coalesce(reservation_id, 0), restriction_id, created_at, updated_at 
		FROM room_restrictions 
		WHERE room_id = $1 AND $2 < end_date AND $3 > start_date`

	rows, err := m.DB.QueryContext(ctx, query, roomID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restriction models.RoomRestriction
		err := rows.Scan(&restriction.ID, &restriction.StartDate, &restriction.EndDate,
			&restriction.RoomID, &restriction.ReservationID,
			&restriction.RestrictionID, &restriction.CreatedAt, &restriction.UpdatedAt)
		if err != nil {
			return nil, err
		}
		restrictions = append(restrictions, restriction)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return restrictions, nil
}

// InsertBlockForRoom inserts a block for a room in the database
func (m *postgresDBRepo) InsertBlockForRoom(id int, startDate time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO room_restrictions (start_date, end_date, room_id, restriction_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := m.DB.ExecContext(ctx, stmt, startDate, startDate.AddDate(0, 0, 1), id, 2, time.Now(), time.Now())
	if err != nil {
		return err
	}

	return nil
}

// DeleteBlockByID deletes a block by its ID
func (m *postgresDBRepo) DeleteBlockByID(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `DELETE FROM room_restrictions WHERE id = $1`

	_, err := m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	return nil
}