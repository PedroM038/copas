package model

type Truque struct {
	CartasJogadas []CartaJogada
	Vencedor      *Jogador
	NaipeTruco    string
}

type CartaJogada struct {
	Jogador *Jogador
	Carta   Carta
}
