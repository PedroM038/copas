import socket
import json
import time
import threading

class NetworkManager:
    def __init__(self, node_index, nodes):
        self.current_node_index = node_index
        self.nodes = nodes
        self.total_nodes = len(nodes)
        self.next_node_index = (node_index + 1) % self.total_nodes
        
        # Socket UDP
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.sock.bind(nodes[node_index])
        
        # Callback para processar mensagens recebidas
        self.message_handler = None
        self.running = True
        
        # Inicia thread de recepção
        self.receive_thread = threading.Thread(target=self._receive_messages, daemon=True)
        self.receive_thread.start()
    
    def set_message_handler(self, handler):
        """Define o callback para processar mensagens recebidas"""
        self.message_handler = handler
    
    def send_message(self, message, target_node):
        """Envia mensagem para um nó específico"""
        target_address = self.nodes[target_node]
        self.sock.sendto(message.encode(), target_address)
        
        # Log da mensagem enviada
        try:
            msg_data = json.loads(message)
            self._log_message("SEND", msg_data.get("type", "UNKNOWN"), target_node, msg_data)
        except:
            self._log_message("SEND", "TOKEN", target_node, message)
    
    def send_to_all(self, message):
        """Envia mensagem para todos os nós do anel"""
        try:
            msg_data = json.loads(message)
            self._log_message("BROADCAST", msg_data.get("type", "UNKNOWN"), None, msg_data)
        except:
            self._log_message("BROADCAST", "UNKNOWN", None, message)
        
        for i in range(self.total_nodes):
            self.send_message(message, i)
    
    def pass_token(self, node_index):
        """Passa o token para outro nó"""
        self.send_message("TOKEN", node_index)
        print(f"🎯 Token passado para Player {node_index}")
    
    def _receive_messages(self):
        """Thread para receber mensagens"""
        while self.running:
            try:
                data, addr = self.sock.recvfrom(1024)
                message = data.decode()
                
                if self.message_handler:
                    self.message_handler(message, addr)
                    
            except socket.error as e:
                if self.running:
                    print(f"⚠️ Erro ao receber mensagem: {e}")
    
    def _log_message(self, action, message_type, target=None, data=None):
        """Log padronizado para mensagens enviadas/recebidas"""
    
    def close(self):
        """Fecha a conexão"""
        self.running = False
        self.sock.close()