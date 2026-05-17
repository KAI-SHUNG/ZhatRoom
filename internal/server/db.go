package server

import (
	"ZhatRoom/internal/protocol"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage struct {
	DB *gorm.DB
}

func InitDB(dsn string) *Storage {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	db.AutoMigrate(&protocol.User{}, &protocol.Message{}, &protocol.RoomModel{})

	s := &Storage{DB: db}
	s.EnsureLobby()
	return s
}

func (s *Storage) NewMessage(msg protocol.Message) error {
	return s.DB.Create(&msg).Error
}

func (s *Storage) GetMessages(roomID uint, limit int, before int64) ([]protocol.Message, error) {
	var msgs []protocol.Message
	q := s.DB.Where("room_id = ?", roomID).Order("created_at DESC").Limit(limit)
	if before > 0 {
		q = q.Where("created_at < ?", before)
	}
	err := q.Find(&msgs).Error
	return msgs, err
}

func (s *Storage) EnsureLobby() {
	s.DB.Exec(`INSERT INTO room_models (id, name, owner_id, created_at) VALUES (0, 'lobby', '', 0) ON CONFLICT DO NOTHING`)
	s.DB.Exec(`ALTER SEQUENCE room_models_id_seq RESTART WITH 1`)
}

func (s *Storage) CreateRoom(name string, ownerID string) (*protocol.RoomModel, error) {
	room := &protocol.RoomModel{Name: name, OwnerID: ownerID}
	if err := s.DB.Create(room).Error; err != nil {
		return nil, fmt.Errorf("room name already exists or DB error: %w", err)
	}
	return room, nil
}

func (s *Storage) GetRoomByName(name string) (*protocol.RoomModel, error) {
	var room protocol.RoomModel
	if err := s.DB.Where("name = ?", name).First(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *Storage) GetRoomByID(id uint) (*protocol.RoomModel, error) {
	var room protocol.RoomModel
	if err := s.DB.Where("id = ?", id).First(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *Storage) ListRooms() ([]protocol.RoomModel, error) {
	var rooms []protocol.RoomModel
	err := s.DB.Order("id ASC").Find(&rooms).Error
	return rooms, err
}

func (s *Storage) DeleteRoom(id uint) error {
	return s.DB.Where("id = ?", id).Delete(&protocol.RoomModel{}).Error
}

func (s *Storage) UserExists(id string) (bool, error) {
	var count int64
	err := s.DB.Model(&protocol.User{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Storage) NewUser(id string, nickname string) error {
	user := protocol.User{
		ID:       id,
		Nickname: nickname,
	}
	return s.DB.Create(&user).Error
}

func (s *Storage) Close() error {
	db, err := s.DB.DB()
	if db == nil {
		return nil
	}
	if err != nil {
		return err
	}
	return db.Close()
}

func (s *Storage) GetUser(id string) (*protocol.User, error) {
	var user protocol.User
	err := s.DB.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Storage) ListUsers() ([]protocol.User, error) {
	var users []protocol.User
	err := s.DB.Find(&users).Error
	return users, err
}

func (s *Storage) DeleteUser(id string) error {
	return s.DB.Where("id = ?", id).Delete(&protocol.User{}).Error
}

func (s *Storage) GetUserByNickname(nickname string) (*protocol.User, error) {
	var user protocol.User
	err := s.DB.Where("nickname = ?", nickname).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Storage) UserExistsByNickname(nickname string) (bool, error) {
	var count int64
	err := s.DB.Model(&protocol.User{}).Where("nickname = ?", nickname).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
