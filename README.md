# Descrição do Projeto

Implementar o jogo copas em uma rede em anel com 4 máquinas

- Criar uma rede em anel com 4 máquinas usando socket DGRAM
- O Controle de acesso a rede deve ser feito por passagem de bastão
- Todas as mensagens devem dar a volta toda pelo anel
- O bastão não é temporizado
- As mensagens não são temporizadas
- Utilizar linguagem Go
- Não é necessário timeout
- O protocolo pode ser de livre escolha

# Regras do Jogo de Copas (Hearts)

## Objetivo do Jogo
- Evitar pegar cartas de penalidade durante as rodadas.
- Cada **♥ (Copas)** vale **1 ponto**.
- A **Q♠ (Dama de Espadas)** vale **13 pontos**.
- O jogo termina quando um jogador atinge **100 pontos ou mais**.
- **Vence quem tiver menos pontos** ao final.

---

## Jogadores
- 4 jogadores.
- Baralho padrão de **52 cartas** (sem coringas).
- Cada jogador recebe **13 cartas**.

---

## Passes (Troca de Cartas)
Antes de cada rodada, cada jogador passa 3 cartas:

| Rodada | Direção do Passe |
|--------|------------------|
| 1ª     | Esquerda         |
| 2ª     | Direita          |
| 3ª     | Frente           |
| 4ª     | Sem passe        |

Depois da 4ª rodada, o ciclo se repete.

---

## Jogando uma Rodada
- O jogador com o **2♣ (Dois de Paus)** começa a rodada.
- Deve obrigatoriamente jogar o **2♣** como primeira carta.
- Os jogadores devem **seguir o naipe da carta inicial** se possível.
- Quem não puder seguir o naipe pode jogar qualquer outra carta (com restrições, veja abaixo).
- A **maior carta do naipe inicial vence a vaza** e começa a próxima.

---

## Regras para Copas
- **Copas não pode ser iniciada (quebrada)** até que alguém jogue uma carta de copas **porque não tinha o naipe da vez**.
- Exceção: se o jogador **só tiver cartas de copas**, pode começar com uma.

---

## Regras para a Dama de Espadas (Q♠)
- Pode ser jogada **somente quando o jogador não puder seguir o naipe da rodada**.
- Pode ser jogada como carta inicial **somente após as copas serem quebradas** (ou se for a única opção do jogador).

---

## Pontuação
- Cada carta de **♥**: **1 ponto**
- **Q♠**: **13 pontos**
- Total de pontos em uma rodada: **26 pontos**

### "Atirar a Lua" (Shoot the Moon)
Se um jogador capturar **todos os 26 pontos**:
- Ele pode escolher:
  - **Zerar sua pontuação**
  - **Adicionar 26 pontos a todos os outros jogadores**

---

## Fim do Jogo
- O jogo termina quando **um jogador chega a 100 pontos ou mais**.
- O jogador com **menos pontos** vence.

---
