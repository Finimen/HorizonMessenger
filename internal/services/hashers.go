package services

import "golang.org/x/crypto/bcrypt"

type BcryptHasher struct {
}

type IHasher interface {
	GenerateFromPassword(password []byte, cost int) ([]byte, error)
	CompareHashAndPassword(storedPaswsord []byte, userPassword []byte) error
	DefaultCost() int
}

func (b *BcryptHasher) DefaultCost() int {
	return bcrypt.DefaultCost
}

func (b *BcryptHasher) GenerateFromPassword(password []byte, cost int) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, cost)
}

func (b *BcryptHasher) CompareHashAndPassword(storedPaswsord []byte, userPassword []byte) error {
	err := bcrypt.CompareHashAndPassword(storedPaswsord, userPassword)

	if err != nil {
		return err
	}

	return nil
}
