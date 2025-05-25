package network

// Token representa o bastão de controle de acesso à rede.
type Token struct {
    HolderID int // ID do jogador que possui o token
}

// NewToken cria um novo token para o jogador inicial.
func NewToken(holderID int) Token {
    return Token{HolderID: holderID}
}

// TokenMessagePayload é o payload usado na mensagem de token.
type TokenMessagePayload struct {
    Token Token `json:"token"`
}

// Cria uma mensagem de passagem de token.
func NewTokenMessage(from, to, holderID int) Message {
    payload := TokenMessagePayload{
        Token: Token{HolderID: holderID},
    }
    return Message{
        Type:    MessageToken,
        From:    from,
        To:      to,
        Payload: payload,
    }
}