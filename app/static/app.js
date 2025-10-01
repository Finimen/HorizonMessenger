// app.js - Refactored with modern architecture
class ChatApp {
    constructor() {
        this.state = {
            ws: null,
            token: null,
            username: null,
            currentChatId: null,
            currentMembers: [],
            userChats: [],
            currentChatInfo: null,
            lastMessagesCache: new Map(),
            stopParticles: null
        };

        this.apiService = new ApiService();
        this.uiManager = new UIManager();
        this.websocketService = new WebSocketService(this);
        this.chatManager = new ChatManager(this);
        this.authManager = new AuthManager(this);
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.checkExistingSession();
    }

    setupEventListeners() {
        // Message input
        document.getElementById('messageInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.chatManager.sendMessage();
        });

        // Send button
        document.getElementById('sendButton').addEventListener('click', () => {
            this.chatManager.sendMessage();
        });

        // Chat title click for info
        document.getElementById('chatTitle').addEventListener('click', () => {
            if (this.state.currentChatInfo) {
                this.uiManager.showChatInfo(this.state.currentChatInfo, this.state.username);
            }
        });
    }

    async checkExistingSession() {
        const token = localStorage.getItem('chatToken');
        const username = localStorage.getItem('chatUsername');
        
        if (token && username) {
            this.state.token = token;
            this.state.username = username;
            
            await this.uiManager.showMessenger(username);
            await this.chatManager.loadUserChats();
            this.websocketService.connect();
        } else {
            this.uiManager.showAuth();
        }

        // Check for email verification token
        this.authManager.handleEmailVerificationFromURL();
    }

    // State management
    setState(newState) {
        this.state = { ...this.state, ...newState };
    }

    logout() {
        this.websocketService.disconnect();
        this.setState({
            token: null,
            username: null,
            currentChatId: null,
            userChats: [],
            currentChatInfo: null
        });
        
        localStorage.removeItem('chatToken');
        localStorage.removeItem('chatUsername');
        this.uiManager.showAuth();
        this.uiManager.clearChatUI();
    }
}

// API Service - handles all HTTP requests
class ApiService {
    constructor() {
        this.baseURL = '/api';
    }

    async request(endpoint, options = {}) {
        const config = {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        };

        try {
            const response = await fetch(`${this.baseURL}${endpoint}`, config);
            const data = await response.json();
            
            return {
                ok: response.ok,
                status: response.status,
                data
            };
        } catch (error) {
            return {
                ok: false,
                error: error.message
            };
        }
    }

    async requestWithAuth(endpoint, options = {}) {
        const token = localStorage.getItem('chatToken');
        if (!token) throw new Error('No authentication token');

        return this.request(endpoint, {
            ...options,
            headers: {
                'Authorization': `Bearer ${token}`,
                ...options.headers
            }
        });
    }

    // Auth endpoints
    async login(credentials) {
        return this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify(credentials)
        });
    }

    async register(userData) {
        return this.request('/auth/register', {
            method: 'POST',
            body: JSON.stringify(userData)
        });
    }

    async resendVerification(identifier) {
        return this.request('/auth/resend-verification', {
            method: 'POST',
            body: JSON.stringify(identifier)
        });
    }

    async verifyEmail(token) {
        return this.request(`/auth/verify-email?token=${token}`);
    }

    async checkVerificationStatus(token) {
        return this.request('/auth/verification-status', {
            headers: { 'Authorization': `Bearer ${token}` }
        });
    }

    async resendVerificationForUser(username) {
        return this.request('/auth/resend-verification', {
            method: 'POST',
            body: JSON.stringify({ username })
        });
    }

    // Chat endpoints
    async getChats() {
        return this.requestWithAuth('/chats');
    }

    async getChatMessages(chatId, limit = 50) {
        return this.requestWithAuth(`/chats/${chatId}/messages?limit=${limit}`);
    }

    async createChat(chatData) {
        return this.requestWithAuth('/chats', {
            method: 'POST',
            body: JSON.stringify(chatData)
        });
    }

    async deleteChat(chatId) {
        return this.requestWithAuth(`/chats/${chatId}`, {
            method: 'DELETE'
        });
    }
}

// UI Manager - handles all DOM manipulations
class UIManager {
    constructor() {
        this.elements = this.cacheDOMElements();
    }

    cacheDOMElements() {
        return {
            // Auth elements
            authSection: document.getElementById('authSection'),
            loginForm: document.getElementById('loginForm'),
            registerForm: document.getElementById('registerForm'),
            verifyEmailSection: document.getElementById('verifyEmailSection'),
            messengerContainer: document.getElementById('messengerContainer'),
            
            // Input fields
            loginUsername: document.getElementById('loginUsername'),
            loginPassword: document.getElementById('loginPassword'),
            registerUsername: document.getElementById('registerUsername'),
            registerEmail: document.getElementById('registerEmail'),
            registerPassword: document.getElementById('registerPassword'),
            
            // Message displays
            loginMessage: document.getElementById('loginMessage'),
            registerMessage: document.getElementById('registerMessage'),
            createChatMessage: document.getElementById('createChatMessage'),
            
            // Chat elements
            currentUser: document.getElementById('currentUser'),
            chatTitle: document.getElementById('chatTitle'),
            contactsScroll: document.getElementById('contactsScroll'),
            chatMessages: document.getElementById('chatMessages'),
            messageInput: document.getElementById('messageInput'),
            sendButton: document.getElementById('sendButton'),
            
            // Modal elements
            createChatModal: document.getElementById('createChatModal'),
            chatInfoModal: document.getElementById('chatInfoModal'),
            chatName: document.getElementById('chatName'),
            memberInput: document.getElementById('memberInput'),
            membersList: document.getElementById('membersList')
        };
    }

    // Auth UI
    showAuth() {
        this.hideElement(this.elements.messengerContainer);
        this.showElement(this.elements.authSection);
    }

    showMessenger(username) {
        this.hideElement(this.elements.authSection);
        this.showElement(this.elements.messengerContainer);
        this.elements.currentUser.textContent = username;
    }

    showLoginForm() {
        this.hideElement(this.elements.registerForm);
        this.showElement(this.elements.loginForm);
        this.clearAuthMessages();
    }

    showRegisterForm() {
        this.hideElement(this.elements.loginForm);
        this.showElement(this.elements.registerForm);
        this.clearAuthMessages();
    }

    showVerifyEmail() {
        this.hideElement(this.elements.loginForm);
        this.hideElement(this.elements.registerForm);
        this.showElement(this.elements.verifyEmailSection);
    }

    // Chat UI
    displayChats(chats, currentUsername, lastMessagesCache, onChatSelect, onChatDelete) {
        const container = this.elements.contactsScroll;
        
        if (chats.length === 0) {
            container.innerHTML = this.safeHTML`
                <div class="no-chats-message">
                    No chats yet
                </div>
            `;
            return;
        }

        const chatsHTML = chats.map(chat => {
            const lastMessage = lastMessagesCache.get(chat.id);
            const displayName = chat.name || 
                `Chat with ${chat.members?.filter(m => m !== currentUsername).join(', ') || 'others'}`;
            
            const lastMessageText = lastMessage ? 
                `${lastMessage.sender}: ${this.truncateText(lastMessage.content, 30)}` : 
                'No messages yet';

            return this.safeHTML`
                <div class="contact" data-chat-id="${chat.id}">
                    <div class="contact-avatar">${displayName.charAt(0)}</div>
                    <div class="contact-info">
                        <h3>${displayName}</h3>
                        <p>${lastMessageText}</p>
                    </div>
                    <button class="delete-chat-btn" 
                            onclick="event.stopPropagation(); app.chatManager.confirmDeleteChat('${chat.id}', '${this.escapeString(displayName)}')">
                        ×
                    </button>
                </div>
            `;
        }).join('');

        container.innerHTML = chatsHTML;

        // Add event listeners
        chats.forEach(chat => {
            const displayName = chat.name || 
                `Chat with ${chat.members?.filter(m => m !== currentUsername).join(', ') || 'others'}`;
            const element = container.querySelector(`[data-chat-id="${chat.id}"]`);
            if (element) {
                element.addEventListener('click', () => onChatSelect(chat.id, displayName, chat));
            }
        });
    }

    displayMessages(messages, currentUsername) {
        const container = this.elements.chatMessages;
        
        if (messages.length === 0) {
            container.innerHTML = this.safeHTML`
                <div class="no-messages">
                    No messages yet
                </div>
            `;
            return;
        }

        // Sort messages chronologically
        const sortedMessages = [...messages].sort((a, b) => 
            new Date(a.timestamp || a.created_at || 0) - new Date(b.timestamp || b.created_at || 0)
        );

        const messagesHTML = sortedMessages.map(message => {
            const isOwn = message.sender === currentUsername;
            const timestamp = this.formatTimestamp(message.timestamp || message.created_at);
            
            return this.safeHTML`
                <div class="message ${isOwn ? 'my-message' : 'other-message'}">
                    <div class="message-header">
                        <span class="message-sender">${message.sender}</span>
                        <span class="message-time">${timestamp}</span>
                    </div>
                    <div class="message-text">${message.content}</div>
                </div>
            `;
        }).join('');

        container.innerHTML = messagesHTML;
        this.scrollToBottom(container);
    }

    updateChatLastMessage(chatId, message, currentUsername) {
        const contactElement = document.querySelector(`[data-chat-id="${chatId}"]`);
        if (!contactElement) return;

        const messageElem = contactElement.querySelector('p');
        const shortContent = this.truncateText(message.content, 30);
        messageElem.textContent = `${message.sender}: ${shortContent}`;
    }

    // Modal management
    showCreateChatModal() {
        this.showElement(this.elements.createChatModal);
    }

    closeCreateChatModal() {
        this.hideElement(this.elements.createChatModal);
        this.elements.chatName.value = '';
        this.elements.membersList.innerHTML = '';
    }

    showChatInfo(chatInfo, currentUsername) {
        const modal = this.elements.chatInfoModal;
        const chatNameElem = document.getElementById('modalChatName');
        const createdElem = document.getElementById('modalChatCreated');
        const membersElem = document.getElementById('modalChatMembers');

        chatNameElem.textContent = chatInfo.name || 'Unnamed Chat';
        
        const createdDate = new Date(chatInfo.created_at);
        createdElem.textContent = createdDate.toLocaleString();
        
        membersElem.innerHTML = '';
        if (chatInfo.members && chatInfo.members.length > 0) {
            chatInfo.members.forEach(member => {
                const li = document.createElement('li');
                li.textContent = member;
                if (member === currentUsername) {
                    li.style.fontWeight = 'bold';
                    li.textContent += ' (You)';
                }
                membersElem.appendChild(li);
            });
        } else {
            membersElem.innerHTML = '<li>No members found</li>';
        }
        
        this.showElement(modal);
    }

    closeChatInfo() {
        this.hideElement(this.elements.chatInfoModal);
    }

    // Utility methods
    showMessage(element, message, type = 'info') {
        element.textContent = message;
        element.className = type;
        setTimeout(() => element.textContent = '', 5000);
    }

    showNotification(message, type = 'info') {
        // In a real app, use a proper notification system
        const notification = type === 'error' ? `❌ ${message}` : `📢 ${message}`;
        alert(notification);
    }

    formatTimestamp(timestamp) {
        try {
            if (!timestamp) return '';
            const date = new Date(timestamp);
            return isNaN(date.getTime()) ? '' : date.toLocaleTimeString([], { 
                hour: '2-digit', 
                minute: '2-digit' 
            });
        } catch (e) {
            return '';
        }
    }

    truncateText(text, maxLength) {
        return text.length > maxLength ? text.substring(0, maxLength) + '...' : text;
    }

    scrollToBottom(element) {
        element.scrollLeft = element.scrollWidth - element.clientWidth;
    }

    safeHTML(strings, ...values) {
        return strings.reduce((result, string, i) => {
            const value = values[i] ?? '';
            return result + string + this.escapeHTML(value);
        }, '');
    }

    escapeHTML(str) {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    escapeString(str) {
        return str.replace(/'/g, "\\'").replace(/"/g, '\\"');
    }

    showElement(element) {
        element.classList.remove('hidden');
    }

    hideElement(element) {
        element.classList.add('hidden');
    }

    clearAuthMessages() {
        this.elements.loginMessage.textContent = '';
        this.elements.registerMessage.textContent = '';
    }

    clearChatUI() {
        this.elements.contactsScroll.innerHTML = '';
        this.elements.chatMessages.innerHTML = '';
        this.elements.chatTitle.textContent = 'Select a chat';
        this.elements.messageInput.placeholder = 'Select a chat to send messages...';
        this.elements.messageInput.disabled = true;
        this.elements.sendButton.disabled = true;
    }

    setChatActive(chatId, chatName) {
        // Remove active class from all contacts
        document.querySelectorAll('.contact').forEach(contact => {
            contact.classList.remove('active');
        });

        // Add active class to selected contact
        const activeContact = document.querySelector(`[data-chat-id="${chatId}"]`);
        if (activeContact) {
            activeContact.classList.add('active');
        }

        // Update chat title
        this.elements.chatTitle.textContent = chatName;
        this.elements.chatTitle.style.cursor = 'pointer';
        this.elements.chatTitle.title = 'Click for chat info';

        // Enable message input
        this.elements.messageInput.placeholder = `Message in ${chatName}...`;
        this.elements.messageInput.disabled = false;
        this.elements.sendButton.disabled = false;
    }
}

// WebSocket Service - handles real-time communication
class WebSocketService {
    constructor(app) {
        this.app = app;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
    }

    connect() {
        if (!this.app.state.token) {
            console.warn('No token available for WebSocket connection');
            return;
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${this.app.state.token}`;
        
        try {
            this.ws = new WebSocket(wsUrl);
            this.setupEventHandlers();
        } catch (error) {
            console.error('Failed to create WebSocket:', error);
            this.scheduleReconnect();
        }
    }

    setupEventHandlers() {
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
            this.app.uiManager.elements.messageInput.disabled = false;
            this.app.uiManager.elements.sendButton.disabled = false;
            
            // Join all user chats
            setTimeout(() => this.joinAllChats(), 500);
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.handleMessage(message);
            } catch (error) {
                console.log('System message:', event.data);
            }
        };

        this.ws.onclose = (event) => {
            console.log('WebSocket disconnected');
            this.app.uiManager.elements.messageInput.disabled = true;
            this.app.uiManager.elements.sendButton.disabled = true;
            this.scheduleReconnect();
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    handleMessage(message) {
        console.log('WebSocket message received:', message);
        
        switch (message.type) {
            case 'message':
                this.app.chatManager.handleNewMessage(message);
                break;
                
            case 'chat_created':
                this.app.chatManager.handleChatCreated(message);
                break;
                
            case 'chat_deleted':
                if (this.app.state.currentChatId === message.chat_id) {
                    this.app.setState({ currentChatId: null, currentChatInfo: null });
                    this.app.uiManager.clearChatUI();
                }
                this.app.chatManager.loadUserChats();
                break;
                
            case 'error':
                this.app.uiManager.showNotification(`Error: ${message.details || message.error}`, 'error');
                break;
        }
    }

    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        } else {
            throw new Error('WebSocket not connected');
        }
    }

    joinChat(chatId) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
        
        this.send({
            type: 'join_chat',
            chat_id: parseInt(chatId),
            sender: this.app.state.username
        });
    }

    joinAllChats() {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            setTimeout(() => this.joinAllChats(), 1000);
            return;
        }

        this.app.state.userChats.forEach(chat => {
            this.joinChat(chat.id);
        });
    }

    scheduleReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = Math.min(1000 * this.reconnectAttempts, 10000);
            console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);
            setTimeout(() => this.connect(), delay);
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
}

// Chat Manager - handles chat-related operations
class ChatManager {
    constructor(app) {
        this.app = app;
    }

    async loadUserChats() {
        try {
            const response = await this.app.apiService.getChats();
            
            if (response.ok) {
                this.app.setState({ 
                    userChats: response.data.chats || [] 
                });
                
                this.app.uiManager.displayChats(
                    this.app.state.userChats,
                    this.app.state.username,
                    this.app.state.lastMessagesCache,
                    (chatId, chatName, chatInfo) => this.selectChat(chatId, chatName, chatInfo),
                    (chatId, chatName) => this.confirmDeleteChat(chatId, chatName)
                );

                // Load last messages for all chats
                this.app.state.userChats.forEach(chat => {
                    if (!this.app.state.lastMessagesCache.has(chat.id)) {
                        this.loadLastMessage(chat.id);
                    }
                });

                // Join all chats via WebSocket
                setTimeout(() => {
                    this.app.websocketService.joinAllChats();
                }, 100);
                
            } else {
                this.app.setState({ userChats: [] });
                this.app.uiManager.displayChats([]);
            }
        } catch (error) {
            console.error('Failed to load user chats:', error);
            this.app.setState({ userChats: [] });
            this.app.uiManager.displayChats([]);
        }
    }

    async loadLastMessage(chatId) {
        try {
            const response = await this.app.apiService.getChatMessages(chatId, 1);
            
            if (response.ok && response.data.messages && response.data.messages.length > 0) {
                const lastMessage = response.data.messages[response.data.messages.length - 1];
                this.app.state.lastMessagesCache.set(chatId, lastMessage);
                
                this.app.uiManager.updateChatLastMessage(chatId, lastMessage, this.app.state.username);
            }
        } catch (error) {
            console.error('Failed to load last message:', error);
        }
    }

    async selectChat(chatId, chatName, chatInfo) {
        this.app.setState({
            currentChatId: chatId,
            currentChatInfo: chatInfo
        });

        this.app.uiManager.setChatActive(chatId, chatName);
        this.app.websocketService.joinChat(chatId);
        await this.loadChatMessages(chatId);
    }

    async loadChatMessages(chatId) {
        try {
            this.app.uiManager.elements.chatMessages.innerHTML = 
                '<div class="loading-messages">Loading messages...</div>';
            
            const response = await this.app.apiService.getChatMessages(chatId);
            
            if (response.ok) {
                this.app.uiManager.displayMessages(
                    response.data.messages || [], 
                    this.app.state.username
                );
            } else {
                throw new Error('Failed to load messages');
            }
        } catch (error) {
            this.app.uiManager.elements.chatMessages.innerHTML = 
                '<div class="error-messages">Failed to load messages</div>';
        }
    }

    handleChatCreated(message) {
        if (message.members && message.members.includes(this.app.state.username)) {
            this.loadUserChats();
        }
    }

    sendMessage() {
        if (!this.app.state.currentChatId) {
            this.app.uiManager.showNotification('Please select a chat first', 'error');
            return;
        }
        
        if (!this.app.websocketService.ws || 
            this.app.websocketService.ws.readyState !== WebSocket.OPEN) {
            this.app.uiManager.showNotification('WebSocket not connected. Please refresh the page.', 'error');
            return;
        }

        const input = this.app.uiManager.elements.messageInput;
        const message = input.value.trim();

        if (!message) {
            this.app.uiManager.showNotification('Message cannot be empty', 'error');
            return;
        }

        const numericChatId = parseInt(this.app.state.currentChatId, 10);
        if (isNaN(numericChatId)) {
            this.app.uiManager.showNotification('Invalid chat ID', 'error');
            return;
        }

        try {
            this.app.websocketService.send({
                type: 'message',
                chat_id: numericChatId,
                content: message,
                sender: this.app.state.username
            });
            input.value = '';
        } catch (error) {
            this.app.uiManager.showNotification(`Failed to send message: ${error.message}`, 'error');
        }
    }

    handleNewMessage(message) {
        if (message.chat_id) {
            // Update last messages cache
            this.app.state.lastMessagesCache.set(message.chat_id, message);
            this.app.uiManager.updateChatLastMessage(message.chat_id, message, this.app.state.username);
        }
        
        // If message is for current chat, display it
        if (this.app.state.currentChatId && message.chat_id === this.app.state.currentChatId) {
            const messagesContainer = this.app.uiManager.elements.chatMessages;
            const isOwn = message.sender === this.app.state.username;
            const currentTime = new Date().toLocaleTimeString([], { 
                hour: '2-digit', 
                minute: '2-digit' 
            });
            
            const messageHTML = this.app.uiManager.safeHTML`
                <div class="message ${isOwn ? 'my-message' : 'other-message'}">
                    <div class="message-header">
                        <span class="message-sender">${message.sender}</span>
                        <span class="message-time">${currentTime}</span>
                    </div>
                    <div class="message-text">${message.content}</div>
                </div>
            `;
            
            messagesContainer.insertAdjacentHTML('beforeend', messageHTML);
            this.app.uiManager.scrollToBottom(messagesContainer);
        }
    }

    async createChat(chatName, members) {
        const messageDiv = this.app.uiManager.elements.createChatMessage;

        if (!chatName.trim()) {
            this.app.uiManager.showMessage(messageDiv, 'Please enter chat name', 'error');
            return false;
        }

        if (members.length === 0) {
            this.app.uiManager.showMessage(messageDiv, 'Please add at least one member', 'error');
            return false;
        }

        try {
            const allMembers = [...members, this.app.state.username];
            
            const response = await this.app.apiService.createChat({
                chat_name: chatName,
                member_ids: allMembers 
            });

            if (response.ok) {
                this.app.uiManager.showMessage(messageDiv, 'Chat created successfully!', 'success');
                
                this.app.uiManager.elements.chatName.value = '';
                this.app.setState({ currentMembers: [] });
                this.app.uiManager.elements.membersList.innerHTML = '';
                
                this.app.uiManager.closeCreateChatModal();
                
                await this.loadUserChats();
                return true;
            } else {
                this.app.uiManager.showMessage(
                    messageDiv, 
                    response.data.error || `Failed to create chat: ${response.status}`, 
                    'error'
                );
                return false;
            }
        } catch (error) {
            this.app.uiManager.showMessage(messageDiv, `Error: ${error.message}`, 'error');
            return false;
        }
    }

    confirmDeleteChat(chatId, chatName) {
        const contactElement = document.querySelector(`[data-chat-id="${chatId}"]`);
        const originalContent = contactElement.innerHTML;
        
        contactElement.innerHTML = this.app.uiManager.safeHTML`
            <div class="delete-confirmation">
                <div class="delete-text">Delete "${chatName}"?</div>
                <div class="confirm-buttons">
                    <button class="btn-confirm-delete" onclick="app.chatManager.deleteChat('${chatId}')">
                        Yes
                    </button>
                    <button class="btn-cancel-delete" 
                            onclick="app.chatManager.cancelDelete('${chatId}', '${this.app.uiManager.escapeString(originalContent)}')">
                        No
                    </button>
                </div>
            </div>
        `;
    }

    cancelDelete(chatId, originalContent) {
        const contactElement = document.querySelector(`[data-chat-id="${chatId}"]`);
        if (contactElement) {
            contactElement.innerHTML = originalContent;
            
            const chat = this.app.state.userChats.find(c => c.id === chatId);
            if (chat) {
                const displayName = chat.name || 
                    `Chat with ${chat.members?.filter(m => m !== this.app.state.username).join(', ') || 'others'}`;
                contactElement.addEventListener('click', () => 
                    this.selectChat(chatId, displayName, chat)
                );
            }
        }
    }

    async deleteChat(chatId) {
        try {
            const response = await this.app.apiService.deleteChat(chatId);

            if (response.ok) {
                this.app.uiManager.showNotification('Chat deleted successfully');
                
                // Update state
                this.app.setState({
                    userChats: this.app.state.userChats.filter(chat => chat.id !== chatId)
                });

                // Update UI
                this.app.uiManager.displayChats(
                    this.app.state.userChats,
                    this.app.state.username,
                    this.app.state.lastMessagesCache,
                    (chatId, chatName, chatInfo) => this.selectChat(chatId, chatName, chatInfo),
                    (chatId, chatName) => this.confirmDeleteChat(chatId, chatName)
                );

                // If deleted chat was current chat, reset UI
                if (this.app.state.currentChatId === chatId) {
                    this.app.setState({ currentChatId: null, currentChatInfo: null });
                    this.app.uiManager.clearChatUI();
                }
            } else {
                throw new Error(`HTTP ${response.status}`);
            }
        } catch (error) {
            this.app.uiManager.showNotification(`Failed to delete chat: ${error.message}`, 'error');
            this.loadUserChats(); // Reload to sync state
        }
    }

    addMember() {
        const memberInput = this.app.uiManager.elements.memberInput;
        const member = memberInput.value.trim();
        
        if (member && !this.app.state.currentMembers.includes(member)) {
            if (member !== this.app.state.username) {
                this.app.state.currentMembers.push(member);
                this.updateMembersList();
            }
            memberInput.value = '';
        }
    }

    updateMembersList() {
        const membersList = this.app.uiManager.elements.membersList;
        membersList.innerHTML = this.app.state.currentMembers.map(member => 
            `<span class="member-tag">${member}</span>`
        ).join('');
    }
}

// Auth Manager - handles authentication operations
class AuthManager {
    constructor(app) {
        this.app = app;
    }

    async login() {
        const username = this.app.uiManager.elements.loginUsername.value;
        const password = this.app.uiManager.elements.loginPassword.value;
        const messageDiv = this.app.uiManager.elements.loginMessage;

        if (!username || !password) {
            this.app.uiManager.showMessage(messageDiv, 'Please fill in all fields', 'error');
            return;
        }

        try {
            const response = await this.app.apiService.login({ username, password });

            if (response.ok) {
                this.app.setState({
                    token: response.data.token,
                    username: username
                });
                
                localStorage.setItem('chatToken', response.data.token);
                localStorage.setItem('chatUsername', username);
                
                await this.app.uiManager.showMessenger(username);
                await this.app.chatManager.loadUserChats();
                
                setTimeout(() => {
                    this.app.websocketService.connect();
                }, 100);

                // Check verification status
                this.checkVerificationStatus(response.data.token);

            } else {
                if (response.data.error && response.data.error.toLowerCase().includes('email not verified')) {
                    this.handleUnverifiedEmail(response.data.error, username, messageDiv);
                } else {
                    this.app.uiManager.showMessage(
                        messageDiv, 
                        response.data.error || 'Login failed', 
                        'error'
                    );
                }
            }
        } catch (error) {
            this.app.uiManager.showMessage(messageDiv, 'Network error: ' + error.message, 'error');
        }
    }

    async register() {
        const username = this.app.uiManager.elements.registerUsername.value;
        const email = this.app.uiManager.elements.registerEmail.value;
        const password = this.app.uiManager.elements.registerPassword.value;
        const messageDiv = this.app.uiManager.elements.registerMessage;

        if (!username || !email || !password) {
            this.app.uiManager.showMessage(messageDiv, 'Please fill in all fields', 'error');
            return;
        }

        try {
            const response = await this.app.apiService.register({ username, password, email });

            if (response.ok) {
                this.app.uiManager.showVerifyEmail();
            } else {
                this.app.uiManager.showMessage(
                    messageDiv, 
                    response.data.error || 'Registration failed', 
                    'error'
                );
            }
        } catch (error) {
            this.app.uiManager.showMessage(messageDiv, 'Network error: ' + error.message, 'error');
        }
    }

    async resendVerification(email) {
        const messageDiv = this.app.uiManager.elements.registerMessage;

        if (!email) {
            this.app.uiManager.showMessage(messageDiv, 'Email is required', 'error');
            return;
        }

        try {
            const response = await this.app.apiService.resendVerification({ email });

            if (response.ok) {
                this.app.uiManager.showMessage(messageDiv, 'Verification email sent!', 'success');
            } else {
                this.app.uiManager.showMessage(
                    messageDiv, 
                    response.data.error || 'Failed to resend verification email', 
                    'error'
                );
            }
        } catch (error) {
            this.app.uiManager.showMessage(messageDiv, 'Network error: ' + error.message, 'error');
        }
    }

    async resendVerificationForUser(username) {
        const messageDiv = this.app.uiManager.elements.loginMessage;
        
        try {
            const response = await this.app.apiService.resendVerification({ username });

            if (response.ok) {
                this.app.uiManager.showMessage(
                    messageDiv, 
                    'Verification email sent! Please check your inbox.', 
                    'success'
                );
            } else {
                this.app.uiManager.showMessage(
                    messageDiv, 
                    response.data.error || 'Failed to resend verification email', 
                    'error'
                );
            }
        } catch (error) {
            this.app.uiManager.showMessage(messageDiv, 'Network error: ' + error.message, 'error');
        }
    }

    async checkVerificationStatus(token) {
        try {
            const response = await this.app.apiService.checkVerificationStatus(token);

            if (response.ok && !response.data.verified) {
                this.app.uiManager.showNotification(
                    'Please verify your email to access all features', 
                    'warning'
                );
            }
        } catch (error) {
            console.error('Failed to check verification status:', error);
        }
    }

    async handleEmailVerificationFromURL() {
        const urlParams = new URLSearchParams(window.location.search);
        const verificationToken = urlParams.get('token');
        
        if (verificationToken) {
            await this.handleEmailVerification(verificationToken);
        }
    }

    async handleEmailVerification(token) {
        try {
            const response = await this.app.apiService.verifyEmail(token);

            if (response.ok) {
                this.app.uiManager.showNotification('Email verified successfully! You can now login.', 'success');
                window.history.replaceState({}, document.title, window.location.pathname);
            } else {
                this.app.uiManager.showNotification(
                    response.data.error || 'Email verification failed', 
                    'error'
                );
            }
        } catch (error) {
            this.app.uiManager.showNotification('Verification failed: ' + error.message, 'error');
        }
    }

    handleUnverifiedEmail(errorMessage, username, messageDiv) {
        this.app.uiManager.showMessage(
            messageDiv, 
            errorMessage + ' Would you like to resend the verification email?', 
            'error'
        );
        
        setTimeout(() => {
            const resendButton = document.createElement('button');
            resendButton.textContent = 'Resend Verification Email';
            resendButton.className = 'auth-button auth-switch';
            resendButton.style.marginTop = '10px';
            resendButton.onclick = () => this.resendVerificationForUser(username);
            messageDiv.appendChild(resendButton);
        }, 100);
    }
}

// Initialize the application
let app;

document.addEventListener('DOMContentLoaded', function() {
    // Hide splash screen
    setTimeout(() => {
        const splashScreen = document.getElementById('splashScreen');
        if (splashScreen) {
            splashScreen.classList.add('hidden');
            setTimeout(() => {
                splashScreen.style.display = 'none';
            }, 800);
        }

        // Initialize app
        app = new ChatApp();
    }, 3000);
});

// Global functions for HTML onclick handlers (legacy support)
function login() { app.authManager.login(); }
function register() { app.authManager.register(); }
function logout() { app.logout(); }
function showLoginForm() { app.uiManager.showLoginForm(); }
function showRegisterForm() { app.uiManager.showRegisterForm(); }
function showCreateChatModal() { app.uiManager.showCreateChatModal(); }
function closeCreateChatModal() { app.uiManager.closeCreateChatModal(); }
function closeChatInfo() { app.uiManager.closeChatInfo(); }
function addMember() { app.chatManager.addMember(); }
function createChat() { 
    app.chatManager.createChat(
        document.getElementById('chatName').value.trim(),
        app.state.currentMembers
    );
}
function resendVerification() {
    app.authManager.resendVerification(document.getElementById('registerEmail').value);
}