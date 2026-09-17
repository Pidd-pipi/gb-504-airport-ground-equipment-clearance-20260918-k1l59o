package dto

type PageQuery struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=200"`
}

func (p *PageQuery) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
}

type RegisterRequest struct {
	Phone    string `json:"phone" binding:"required,max=20"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Name     string `json:"name" binding:"required,max=50"`
	Role     string `json:"role" binding:"required,oneof=admin safety_manager inspector worker"`
}

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  any    `json:"user"`
}

type UpdateProfileRequest struct {
	Name string `json:"name" binding:"required,max=50"`
}
