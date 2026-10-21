package domain

type Repository interface {
	Category() CategoryRepository
	Expense() ExpenseRepository
	TelegramBindCode() TelegramBindCodeRepository
	User() UserRepository
	UserTelegramContext() UserTelegramContextRepository
}
