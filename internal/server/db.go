package server

import (
	"ZhatRoom/internal/protocol"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage struct {
	DB *gorm.DB
}

func InitDB() *Storage {
	dsn := "host=127.0.0.1 user=postgres dbname=zhat_db password=zhatroom port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	db.AutoMigrate(&protocol.User{}, &protocol.Message{})

	return &Storage{DB: db}
}

func (s *Storage) NewMessage(msg protocol.Message) error {
	return s.DB.Create(&msg).Error
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
