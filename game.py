import random
import time
import sys
import signal
import json
from network import NetworkManager

# Configuração dos nós
nodes = [
    ('localhost', 5000),
    ('localhost', 5001),
    ('localhost', 5002),
    ('localhost', 5003),
]

class HeartsGame:
    def __init__(self, node_index):
        self.current_node_index = node_index
        self.network = NetworkManager(node_index, nodes)
        self.network.set_message_handler(self.handle_message)
        
        # Variáveis do jogo
        self.game_over = False
        self.game_winner = None
        self.current_round = 0
        self.current_trick = 1
        self.current_trick_cards = []
        self.player_hand = []
        self.players_scores = [0, 0, 0, 0]
        self.token = False
        self.hearts_broken = False
        self.trick_starter = None
        self.current_trick_suit = None
        self.first_trick = True
        self.game_started = False
        self.connected_players = set()
        self.all_hands = []

    # PROCESSAMENTO DE MENSAGENS
    def handle_message(self, message, addr):
        # Se for apenas o token, processa diretamente
        if message == "TOKEN":
            self.token = True
            print(f"🎯 Token recebido! É sua vez de jogar.")
            return
        
        # Tenta processar como JSON
        try:
            data = json.loads(message)
            msg_type = data.get("type")
            
            if msg_type == "CONNECT":
                self.process_connect_message(data)
            elif msg_type == "START_GAME":
                self.process_start_game_message(data)
            elif msg_type == "GAME":
                self.process_game_message(data)
            elif msg_type == "END_TRICK":
                self.process_end_trick_message(data)
            elif msg_type == "SCORES":
                self.process_scores_message(data)
            elif msg_type == "NEW_HAND":
                self.process_new_hand_message(data)
            elif msg_type == "GAME_END":
                self.process_game_end_message(data)
            else:
                print(f"⚠️ Tipo de mensagem desconhecido: {msg_type}")
        
        except json.JSONDecodeError:
            print(f"⚠️ Erro ao decodificar JSON: {message}")

    def process_connect_message(self, data):
        player_id = data.get("player")
        
        # Apenas o host (player 0) processa conexões
        if self.current_node_index == 0:
            self.connected_players.add(player_id)
            print(f"🔗 Player {player_id} conectado! ({len(self.connected_players)}/4)")
            
            # Se todos os 4 jogadores estão conectados, inicia o jogo
            if len(self.connected_players) == 4:
                print("🎉 Todos os jogadores conectados! Iniciando jogo...")
                self.start_game_as_host()

    def process_start_game_message(self, data):
        hands = data.get("hands", [])
        
        if self.current_node_index < len(hands):
            self.player_hand = hands[self.current_node_index]
            self.game_started = True
                    
            # Verifica se este jogador tem o 2♣ e deve começar
            if "2♣" in self.player_hand:
                self.network.send_message("TOKEN", self.current_node_index)
            else:
                self.token = False
                print(f"🔄 Player {self.current_node_index} não tem o 2♣, esperando o próximo jogador.")

    def process_game_message(self, data):
        action = data.get("action")
        
        if action == "PLAY":
            card = data.get("card")
            player = data.get("player")
            
            # Adiciona a carta jogada às cartas da rodada atual
            self.current_trick_cards.append({
                "card": card,
                "player": player
            })
            
            print(f"📋 Cartas jogadas no trick atual: {len(self.current_trick_cards)}")

            # Se é a primeira carta da rodada, define o naipe da rodada e quem começou
            if len(self.current_trick_cards) == 1:
                self.current_trick_suit = self.get_card_suit(card)
                self.trick_starter = player
            
            print(f"🃏 Player {player} jogou: {card}")
            print(f"📋 Cartas na mesa: {[c['card'] for c in self.current_trick_cards]}")
            
            # Se todos os 4 jogadores jogaram, termina a rodada
            if len(self.current_trick_cards) == 4:
                self.end_trick()

    def process_end_trick_message(self, data):
        winner = data.get("winner")
        points = data.get("points")
        scores = data.get("scores")
        
        print(f"\n🏆 Player {winner} ganhou a rodada!")
        print(f"📊 Pontos da rodada: {points}")
        
        # Atualiza pontuações
        if scores:
            self.players_scores = scores.copy()
            print(f"📈 Pontuações atuais: {self.players_scores}")
        
        # Limpa as cartas da mesa
        self.current_trick_cards = []
        self.first_trick = False
        
        # Verifica se completou uma mão (13 rodadas)
        if self.current_trick >= 13:
            self.check_game_end()

    def process_scores_message(self, data):
        scores = data.get("scores")
        if scores:
            self.players_scores = scores.copy()
            print(f"📊 Pontuações atualizadas: {self.players_scores}")

    def process_new_hand_message(self, data):
        hands = data.get("hands", [])
        
        if self.current_node_index < len(hands):
            self.player_hand = hands[self.current_node_index]
            self.current_trick = 0
            self.first_trick = True
            self.hearts_broken = False
            self.token = False
            
            print(f"🆕 Nova mão iniciada! Suas cartas: {sorted(self.player_hand, key=lambda x: (self.get_card_suit(x), self.get_card_value(x)))}")
            
            # Verifica se este jogador tem o 2♣ e deve começar
            if "2♣" in self.player_hand:
                self.network.send_message("TOKEN", self.current_node_index)
            else:
                self.token = False
                print(f"🔄 Player {self.current_node_index} não tem o 2♣, esperando o próximo jogador.")

    def process_game_end_message(self, data):
        winner = data.get("winner")
        final_scores = data.get("final_scores")
        
        self.game_over = True
        self.game_winner = winner
        
        print(f"\n🎉 JOGO TERMINADO!")
        print(f"🏆 Player {winner} venceu com {final_scores[winner]} pontos!")
        print(f"📊 Pontuações finais: {final_scores}")
        
        # Encerra o programa após alguns segundos
        time.sleep(3)
        self.network.close()
        sys.exit(0)

    # LÓGICA DO JOGO
    def end_trick(self):
        # Calcula quem ganhou a rodada
        winner = self.get_trick_winner(self.current_trick_cards)
        
        # Calcula pontos da rodada
        points = self.calculate_trick_points(self.current_trick_cards)
        
        # Adiciona pontos ao vencedor
        self.players_scores[winner] += points
        
        print(f"\n🏆 Rodada {self.current_trick} finalizada!")
        print(f"📋 Cartas jogadas: {[card_info['card'] for card_info in self.current_trick_cards]}")
        print(f"🎯 Vencedor: Player {winner}")
        print(f"💔 Pontos: {points}")
        
        self.current_trick += 1
        
        # Envia resultado para todos os jogadores
        end_trick_message = {
            "type": "END_TRICK",
            "winner": winner,
            "points": points,
            "scores": self.players_scores.copy(),
            "trick": self.current_trick + 1
        }
        # se for o ganhador, envia mensagem para todo mundo
        if self.current_node_index == winner:
            self.network.send_to_all(json.dumps(end_trick_message))

    @staticmethod
    def create_deck():
        suits = ['♠', '♥', '♣', '♦']
        values = ['2', '3', '4', '5', '6', '7', '8', '9', '10', 'J', 'Q', 'K', 'A']
        deck = [f"{v}{s}" for s in suits for v in values]
        random.shuffle(deck)
        return deck

    def deal_cards(self):
        deck = self.create_deck()
        hands = [deck[i:i + 13] for i in range(0, 52, 13)]
        return hands

    @staticmethod
    def get_card_value(card):
        value = card[:-1]
        values = {'2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, 
                  '9': 9, '10': 10, 'J': 11, 'Q': 12, 'K': 13, 'A': 14}
        return values[value]

    @staticmethod
    def get_card_suit(card):
        return card[-1]

    @staticmethod
    def get_card_number_value(card):
        """Converte carta para valor numérico para display"""
        value = card[:-1]
        if value == 'J':
            return 11
        elif value == 'Q':
            return 12
        elif value == 'K':
            return 13
        elif value == 'A':
            return 14
        else:
            return int(value)

    def find_card_by_number(self, number, valid_cards):
        """Encontra carta na mão pelo número digitado"""
        for card in valid_cards:
            if self.get_card_number_value(card) == number:
                return card
        return None

    def display_cards_with_numbers(self, cards):
        """Exibe cartas com seus números correspondentes"""
        print("\n🃏 Suas cartas:")
        for i, card in enumerate(sorted(cards, key=lambda x: (self.get_card_suit(x), self.get_card_value(x)))):
            number = self.get_card_number_value(card)
            print(f"   {number}: {card}")

    def is_valid_play(self, card, trick_cards, player_hand):
        # Primeira jogada de todas deve ser 2♣
        if self.first_trick and len(trick_cards) == 0:
            return card == "2♣"
        
        # Na primeira rodada, não pode jogar ♥ ou Q♠
        if self.first_trick:
            if self.get_card_suit(card) == '♥' or card == 'Q♠':
                # Só pode se não tiver outra opção
                other_cards = [c for c in player_hand if self.get_card_suit(c) != '♥' and c != 'Q♠']
                if len(trick_cards) == 0:
                    return len(other_cards) == 0
                else:
                    lead_suit = self.get_card_suit(trick_cards[0]["card"])
                    same_suit_cards = [c for c in other_cards if self.get_card_suit(c) == lead_suit]
                    return len(same_suit_cards) == 0
        
        # Se é o primeiro a jogar na rodada
        if len(trick_cards) == 0:
            # Copas não pode ser jogado até ser quebrado (exceto se só tiver copas)
            if self.get_card_suit(card) == '♥' and not self.hearts_broken:
                non_hearts = [c for c in player_hand if self.get_card_suit(c) != '♥']
                return len(non_hearts) == 0
            return True
        
        # Deve seguir o naipe se tiver
        lead_suit = self.get_card_suit(trick_cards[0]["card"])
        same_suit_cards = [c for c in player_hand if self.get_card_suit(c) == lead_suit]
        if same_suit_cards:
            return self.get_card_suit(card) == lead_suit
        
        return True

    @staticmethod
    def get_card_points(card):
        if HeartsGame.get_card_suit(card) == '♥':
            return 1
        elif card == 'Q♠':
            return 13
        return 0

    def calculate_trick_points(self, trick_cards):
        points = 0
        for card_info in trick_cards:
            points += self.get_card_points(card_info["card"])
        return points

    def get_trick_winner(self, trick_cards):
        lead_suit = self.get_card_suit(trick_cards[0]["card"])
        highest_value = -1
        winner_player = trick_cards[0]["player"]
        
        for card_info in trick_cards:
            card = card_info["card"]
            player = card_info["player"]
            if self.get_card_suit(card) == lead_suit:
                value = self.get_card_value(card)
                if value > highest_value:
                    highest_value = value
                    winner_player = player
        
        return winner_player

    def play_card(self, card):
        if card in self.player_hand:
            self.player_hand.remove(card)
            if self.get_card_suit(card) == '♥':
                self.hearts_broken = True
            
            message = {
                "type": "GAME",
                "action": "PLAY",
                "card": card,
                "player": self.current_node_index
            }
            self.network.send_to_all(json.dumps(message))
            print(f"🃏 Você jogou: {card}")
            self.network.pass_token(self.network.next_node_index)
            self.token = False
        else:        
            print("❌ Você não tem essa carta na mão!")
            return

    def check_game_end(self):
        max_score = max(self.players_scores)
        if max_score >= 100:
            self.game_over = True
            min_score = min(self.players_scores)
            self.game_winner = self.players_scores.index(min_score)
            
            # Envia mensagem de fim de jogo para todos
            game_end_message = {
                "type": "GAME_END",
                "winner": self.game_winner,
                "final_scores": self.players_scores.copy()
            }
            self.network.send_to_all(json.dumps(game_end_message))
            
            print(f"\n🎉 JOGO TERMINADO!")
            print(f"🏆 Player {self.game_winner} venceu com {min_score} pontos!")
            print(f"📊 Pontuações finais: {self.players_scores}")
        else:
            # Se o jogo não acabou mas completou uma mão (13 tricks), inicia nova mão
            print(f"\n🔄 Mão completada! Pontuações atuais: {self.players_scores}")
            if max_score < 100:
                self.start_new_hand()

    def start_new_hand(self):
        print("🆕 Iniciando nova mão...")
        self.current_trick = 0
        self.first_trick = True
        self.hearts_broken = False
        
        if self.current_node_index == 0:  # Host redistribui as cartas
            self.all_hands = self.deal_cards()
            self.player_hand = self.all_hands[0]
            
            start_message = {
                "type": "NEW_HAND",
                "hands": self.all_hands
            }
            self.network.send_to_all(json.dumps(start_message))
        
        if "2♣" in self.player_hand:
            self.network.send_message("TOKEN", self.current_node_index)
        else:
            self.token = False
            print(f"🔄 Player {self.current_node_index} não tem o 2♣, esperando o próximo jogador.")

    # Conexão e início do jogo
    def start_game_as_host(self):
        print("🎲 Host distribuindo cartas...")
        # Gera e distribui as cartas
        self.all_hands = self.deal_cards()
        self.player_hand = self.all_hands[0]  # Host pega a primeira mão
        
        # Envia as cartas para todos os jogadores
        start_message = {
            "type": "START_GAME",
            "hands": self.all_hands
        }
        self.network.send_to_all(json.dumps(start_message))
        
        self.game_started = True
        print(f"🎮 Jogo iniciado! Suas cartas: {self.player_hand}")
        
        # O jogador com 2♣ recebe o token
        if "2♣" in self.player_hand:
            self.network.send_message("TOKEN", self.current_node_index)
        else:
            self.token = False
            print(f"🔄 Player {self.current_node_index} não tem o 2♣, esperando o próximo jogador.")

    def initialize_connection(self):
        self.current_round = 0
        self.game_over = False
        self.hearts_broken = False
        self.first_trick = True
        
        if self.current_node_index == 0:
            print(f"🎮 Player {self.current_node_index} (Host) - Aguardando outros jogadores...")
            self.connected_players.add(0)  # Host está conectado
        else:
            print(f"🎮 Player {self.current_node_index} - Conectando ao jogo...")
            self.announce_connection()
            print("⏳ Aguardando início do jogo...")

    def announce_connection(self):
        """Anuncia conexão para o host"""
        if self.current_node_index != 0:
            connect_message = {
                "type": "CONNECT",
                "player": self.current_node_index
            }
            self.network.send_message(json.dumps(connect_message), 0)  # Envia para o host
            print("📡 Conexão anunciada para o host")

    # Loop principal do jogo
    def run(self):
        print(f"🚀 Iniciando Player {self.current_node_index}")
        self.initialize_connection()
        
        # Aguarda o jogo começar
        while not self.game_started and not self.game_over:
            time.sleep(0.5)
        
        # Loop principal do jogo
        while not self.game_over:
            if self.token:
                print(f"\n{'='*50}")
                print(f"🎯 SUA VEZ! (Player {self.current_node_index})")
                
                self.display_cards_with_numbers(self.player_hand)
                
                valid_cards = [c for c in self.player_hand if self.is_valid_play(c, self.current_trick_cards, self.player_hand)]
                
                if valid_cards:
                    print(f"\n✅ Cartas válidas para jogar:")
                    for card in sorted(valid_cards, key=lambda x: (self.get_card_suit(x), self.get_card_value(x))):
                        number = self.get_card_number_value(card)
                        print(f"   {number}: {card}")
                    
                    if self.current_trick_cards:
                        print(f"\n🃏 Cartas na mesa: {[card_info['card'] for card_info in self.current_trick_cards]}")
                    
                    try:
                        choice = int(input("\n🎯 Digite o número da carta para jogar: "))
                        selected_card = self.find_card_by_number(choice, valid_cards)
                        
                        if selected_card:
                            self.play_card(selected_card)
                        else:
                            print("❌ Número inválido! Tente novamente.")
                    except ValueError:
                        print("❌ Digite apenas números!")
                else:
                    print("⚠️ Nenhuma carta válida disponível!")
            
            time.sleep(0.1)

def signal_handler(sig, frame):
    print("\n🛑 Encerrando...")
    game.network.close()
    sys.exit(0)

if __name__ == "__main__":
    current_node_index = int(sys.argv[1]) if len(sys.argv) > 1 else 0
    game = HeartsGame(current_node_index)
    
    signal.signal(signal.SIGINT, signal_handler)
    game.run()