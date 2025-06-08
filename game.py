import random
import time
import sys
import signal
from network import NetworkManager
from protocol import Protocol

class HeartsGame:
    def __init__(self, node_index, nodes):
        self.player_index = node_index
        
        # Configuração de rede
        self.network = NetworkManager(node_index, nodes)
        self.protocol = Protocol(self.network, self)
        
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
        self.first_trick = True
        self.game_started = False
        self.connected_players = set()

    # MÉTODOS DE ESTADO DO JOGO
    def is_host(self):
        """Verifica se é o host"""
        return self.player_index == 0

    def add_connected_player(self, player_id):
        """Adiciona jogador conectado"""
        self.connected_players.add(player_id)
        if self.is_host():
            self.connected_players.add(0)  # Host sempre está conectado
        return len(self.connected_players)

    def all_players_connected(self):
        """Verifica se todos os jogadores estão conectados"""
        return len(self.connected_players) == 4

    def set_player_hand(self, hand):
        """Define a mão do jogador"""
        self.player_hand = hand

    def start_game(self):
        """Marca o jogo como iniciado"""
        self.game_started = True

    def has_two_of_clubs(self):
        """Verifica se o jogador tem o 2♣"""
        return "2♣" in self.player_hand

    def add_card_to_trick(self, card, player):
        """Adiciona carta à rodada atual"""
        self.current_trick_cards.append({
            "card": card,
            "player": player
        })

    def get_current_trick_cards(self):
        """Retorna lista das cartas na mesa"""
        return [c['card'] for c in self.current_trick_cards]

    def is_trick_complete(self):
        """Verifica se a rodada está completa"""
        return len(self.current_trick_cards) == 4

    def reset_trick(self):
        """Reseta para próxima rodada"""
        self.current_trick_cards = []
        self.first_trick = False

    def is_hand_complete(self):
        """Verifica se a mão está completa (13 rodadas)"""
        return self.current_trick >= 13

    def update_scores(self, scores):
        """Atualiza pontuações"""
        self.players_scores = scores.copy()

    def start_new_hand(self, hand):
        """Inicia nova mão"""
        self.player_hand = hand
        self.current_trick = 0
        self.first_trick = True
        self.hearts_broken = False
        self.token = False

    def end_game(self, winner, final_scores):
        """Finaliza o jogo"""
        self.game_over = True
        self.game_winner = winner
        self.players_scores = final_scores.copy()

    def receive_token(self):
        """Recebe o token (evita duplicação)"""
        if not self.token:  # Só aceita se não tiver já
            self.token = True
            return True
        return False

    def can_play_card(self, card):
        """Verifica se pode jogar a carta"""
        return card in self.player_hand

    def remove_card_from_hand(self, card):
        """Remove carta da mão"""
        if card in self.player_hand:
            self.player_hand.remove(card)
            if self.get_card_suit(card) == '♥':
                self.hearts_broken = True
            return True
        return False

    def calculate_trick_result(self):
        """Calcula resultado da rodada"""
        winner = self.get_trick_winner(self.current_trick_cards)
        points = self.calculate_trick_points(self.current_trick_cards)
        self.players_scores[winner] += points
        return winner, points

    def next_trick(self):
        """Avança para próxima rodada"""
        self.current_trick += 1

    def initialize(self):
        """Inicializa estado do jogo"""
        self.current_round = 0
        self.game_over = False
        self.hearts_broken = False
        self.first_trick = True

    def is_last_player_in_trick(self):
        """Verifica se é o último jogador a jogar no trick atual"""
        return len(self.current_trick_cards) == 3  # 3 cartas já jogadas, falta 1

    def will_complete_trick(self):
        """Verifica se a próxima jogada completará o trick"""
        return len(self.current_trick_cards) == 3

    # LÓGICA DE CARTAS E REGRAS
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
        """Encontra carta na mão pelo número sequencial digitado"""
        sorted_valid = sorted(valid_cards, key=lambda x: (self.get_card_suit(x), self.get_card_value(x)))
        
        if 1 <= number <= len(sorted_valid):
            return sorted_valid[number - 1]
        return None

    def display_cards_with_numbers(self, cards):
        """Exibe cartas com seus números únicos para seleção"""
        print("\n🃏 Suas cartas:")
        sorted_cards = sorted(cards, key=lambda x: (self.get_card_suit(x), self.get_card_value(x)))
        
        for i, card in enumerate(sorted_cards, 1):
            print(f"   {i}: {card}")
    
    def display_valid_cards(self, valid_cards):
        """Exibe cartas válidas com numeração sequencial"""
        print(f"\n✅ Cartas válidas para jogar:")
        sorted_valid = sorted(valid_cards, key=lambda x: (self.get_card_suit(x), self.get_card_value(x)))
        
        for i, card in enumerate(sorted_valid, 1):
            print(f"   {i}: {card}")

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
        """Determina vencedor do trick (maior carta do naipe inicial)"""
        if not trick_cards:
            return None
            
        lead_suit = self.get_card_suit(trick_cards[0]["card"])
        winner_player = None
        highest_value = -1
        
        for card_info in trick_cards:
            card = card_info["card"]
            player = card_info["player"]
            
            # Só cartas do naipe inicial podem ganhar
            if self.get_card_suit(card) == lead_suit:
                value = self.get_card_value(card)
                if value > highest_value:
                    highest_value = value
                    winner_player = player
        
        return winner_player

    def check_game_end(self):
        max_score = max(self.players_scores)
        if max_score >= 100:
            self.game_over = True
            min_score = min(self.players_scores)
            self.game_winner = self.players_scores.index(min_score)
            
            self.protocol.send_game_end_message(self.game_winner, self.players_scores.copy())
            
            print(f"\n🎉 JOGO TERMINADO!")
            print(f"🏆 Player {self.game_winner} venceu com {min_score} pontos!")
            print(f"📊 Pontuações finais: {self.players_scores}")
        else:
            # Se o jogo não acabou mas completou uma mão (13 tricks), inicia nova mão
            print(f"\n🔄 Mão completada! Pontuações atuais: {self.players_scores}")
            if max_score < 100:
                self.start_new_hand_logic()

    def start_new_hand_logic(self):
        print("🆕 Iniciando nova mão...")
        self.current_trick = 0
        self.first_trick = True
        self.hearts_broken = False
        
        if self.is_host():  # Host redistribui as cartas
            hands = self.deal_cards()
            self.player_hand = hands[0]
            self.protocol.send_new_hand_message(hands)
        
        if self.has_two_of_clubs():
            self.protocol.send_token_to_self()
        else:
            self.token = False
            print(f"🔄 Player {self.player_index} não tem o 2♣, esperando o próximo jogador.")

    # Atualizar o loop principal do jogo
    def run(self):
        print(f"🚀 Iniciando Player {self.player_index}")
        self.protocol.initialize_connection()
        
        # Aguarda o jogo começar
        while not self.game_started and not self.game_over:
            time.sleep(0.5)
        
        # Loop principal do jogo
        while not self.game_over:
            if self.token:
                print(f"\n{'='*50}")
                print(f"🎯 SUA VEZ! (Player {self.player_index})")
                
                # Mostra cartas na mesa se houver
                if self.current_trick_cards:
                    print(f"\n🃏 Cartas na mesa: {[card_info['card'] for card_info in self.current_trick_cards]}")
                
                # Exibe todas as cartas
                self.display_cards_with_numbers(self.player_hand)
                
                valid_cards = [c for c in self.player_hand if self.is_valid_play(c, self.current_trick_cards, self.player_hand)]
                
                if valid_cards:
                    self.display_valid_cards(valid_cards)
                    
                    try:
                        choice = int(input(f"\n🎯 Digite o número da carta (1-{len(valid_cards)}): "))
                        selected_card = self.find_card_by_number(choice, valid_cards)
                        
                        if selected_card:
                            self.protocol.play_card(selected_card)
                        else:
                            print(f"❌ Número inválido! Digite entre 1 e {len(valid_cards)}.")
                    except ValueError:
                        print("❌ Digite apenas números!")
                else:
                    print("⚠️ Nenhuma carta válida disponível!")
            
            time.sleep(0.1)

def parse_arguments():
    """Analisa argumentos da linha de comando"""
    if len(sys.argv) < 3:
        print("❌ Uso: python game.py <id_jogador> <endereco> [porta_base]")
        print("📝 Exemplos:")
        print("   python game.py 0 localhost")
        print("   python game.py 1 192.168.1.100")
        print("   python game.py 2 localhost 6000")
        sys.exit(1)
    
    try:
        player_id = int(sys.argv[1])
        address = sys.argv[2]
        base_port = int(sys.argv[3]) if len(sys.argv) > 3 else 5000
        
        if not (0 <= player_id <= 3):
            print("❌ ID do jogador deve ser entre 0 e 3")
            sys.exit(1)
            
        return player_id, address, base_port
        
    except ValueError:
        print("❌ ID do jogador e porta devem ser números")
        sys.exit(1)

def create_nodes_config(player_id, player_address, base_port):
    """Cria configuração dos nós baseada nos argumentos"""
    # Configuração padrão das portas para cada jogador
    ports = [base_port, base_port + 1, base_port + 2, base_port + 3]
    
    # Cria lista de nós
    nodes = []
    for i in range(4):
        if i == player_id:
            # Usa o endereço fornecido para este jogador
            nodes.append((player_address, ports[i]))
        else:
            # Para outros jogadores, assume localhost (pode ser modificado depois)
            nodes.append(('localhost', ports[i]))
    
    return nodes

def signal_handler(sig, frame):
    print("\n🛑 Encerrando...")
    game.network.close()
    sys.exit(0)

if __name__ == "__main__":
    # Analisa argumentos da linha de comando
    player_id, player_address, base_port = parse_arguments()
    
    # Cria configuração dos nós
    nodes = create_nodes_config(player_id, player_address, base_port)
    
    print(f"🚀 Configuração do jogo:")
    print(f"   🎮 Jogador: {player_id}")
    print(f"   🌐 Endereço: {player_address}:{base_port + player_id}")
    
    # Cria e executa o jogo
    game = HeartsGame(player_id, nodes)
    
    signal.signal(signal.SIGINT, signal_handler)
    game.run()