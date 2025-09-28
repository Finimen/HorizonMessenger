let ws = null;
        let currentToken = null;
        let currentUsername = null;
        let currentChatId = null;
        let currentMembers = [];
        let userChats = [];

        async function login() {
            const username = document.getElementById('loginUsername').value;
            const password = document.getElementById('loginPassword').value;
            const messageDiv = document.getElementById('loginMessage');

            if (!username || !password) {
                showMessage(messageDiv, 'Please fill in all fields', 'error');
                return;
            }

            try {
                const response = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, password })
                });

                const data = await response.json();

                if (response.ok) {
                    currentToken = data.token;
                    currentUsername = username;
                    
                    localStorage.setItem('chatToken', currentToken);
                    localStorage.setItem('chatUsername', currentUsername);
                    
                    showChat();
                    loadUserChats();
                    
                    setTimeout(() => {
                        connectWebSocket();
                    }, 100);
            
        } else {
            showMessage(messageDiv, data.error || 'Login failed', 'error');
        }
    } catch (error) {
        showMessage(messageDiv, 'Network error: ' + error.message, 'error');
    }
}

async function register() {
    const username = document.getElementById('registerUsername').value;
    const email = document.getElementById('registerEmail').value;
    const password = document.getElementById('registerPassword').value;
    const messageDiv = document.getElementById('registerMessage');

    if (!username || !email || !password) {
        showMessage(messageDiv, 'Please fill in all fields', 'error');
        return;
    }

    try {
        const response = await fetch('/api/auth/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password, email })
        });

        const data = await response.json();

        if (response.ok) {
            showMessage(messageDiv, 'Registration successful! Please login.', 'success');
            showLoginForm();
        } else {
            showMessage(messageDiv, data.error || 'Registration failed', 'error');
        }
    } catch (error) {
        showMessage(messageDiv, 'Network error: ' + error.message, 'error');
    }
}

function logout() {
    if (ws) {
        ws.close();
    }
    currentToken = null;
    currentUsername = null;
    currentChatId = null;
    showAuth();
    document.getElementById('messages').innerHTML = '';
    document.getElementById('chatList').innerHTML = '';
}

async function loadChatMessages(chatId) {
    try {
        const messagesDiv = document.getElementById('messages');
        messagesDiv.innerHTML = '<div style="text-align: center; color: #666;">Loading messages...</div>';
        
        const response = await fetch(`/api/chats/${chatId}/messages?limit=50`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });

        if (response.ok) {
            const data = await response.json();
            displayMessages(data.messages || []);
        } else if (response.status === 404) {
            messagesDiv.innerHTML = '<div style="text-align: center; color: #666;">Chat not found</div>';
        } else {
            throw new Error(`HTTP ${response.status}`);
        }
    } catch (error) {
        console.error('Failed to load messages:', error);
        const messagesDiv = document.getElementById('messages');
        messagesDiv.innerHTML = '<div style="text-align: center; color: #dc3545;">Failed to load messages</div>';
    }
}

function joinAllChats() {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
    console.log('WebSocket not ready, retrying in 1 second...');
    setTimeout(joinAllChats, 1000);
    return;
}

console.log('Joining all chats:', userChats.length);
userChats.forEach(chat => {
    const joinMessage = {
        type: 'join_chat',
        chat_id: chat.id,
        sender: currentUsername
    };
    ws.send(JSON.stringify(joinMessage));
    console.log('Joined chat:', chat.id, chat.name);
    });
}

function displayChats(chats) {
    const chatList = document.getElementById('chatList');
    chatList.innerHTML = '';

    console.log('Displaying chats:', chats);

    if (chats.length === 0) {
        chatList.innerHTML = '<div style="text-align: center; color: #666;">No chats yet</div>';
        return;
    }

    chats.forEach(chat => {
        const chatItem = document.createElement('div');
        chatItem.className = 'chat-item';
        chatItem.dataset.chatId = chat.id;
        
        const chatName = document.createElement('div');
        chatName.textContent = chat.name || `Chat with ${chat.members?.join(', ') || 'others'}`;
        chatItem.appendChild(chatName);
        
        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'delete-chat-btn';
        deleteBtn.innerHTML = '×';
        deleteBtn.title = 'Delete chat';
        deleteBtn.onclick = (e) => {
            e.stopPropagation();
            confirmDeleteChat(chat.id, chat.name);
        };
        chatItem.appendChild(deleteBtn);
        
        chatItem.onclick = () => selectChat(chat.id, chat.name);
        chatList.appendChild(chatItem);
    });
}

function confirmDeleteChat(chatId, chatName) {
    const chatList = document.getElementById('chatList');
    const chatItem = chatList.querySelector(`[data-chat-id="${chatId}"]`);
    
    if (chatItem.classList.contains('confirm-delete')) {
        return; 
    }
    
    const originalContent = chatItem.innerHTML;
    const originalClasses = chatItem.className;
    
    chatItem.className = 'chat-item confirm-delete';
    chatItem.innerHTML = `
        <div>Delete "${chatName}"?</div>
        <div class="confirm-buttons">
            <button onclick="deleteChat('${chatId}')" style="background-color: #dc3545;">Yes</button>
            <button onclick="cancelDelete('${chatId}', \`${originalContent.replace(/`/g, '\\`')}\`, '${originalClasses}')">No</button>
        </div>
    `;
}

function cancelDelete(chatId, originalContent, originalClasses) {
    const chatItem = document.getElementById('chatList').querySelector(`[data-chat-id="${chatId}"]`);
    chatItem.className = originalClasses;
    chatItem.innerHTML = originalContent;
    
    const deleteBtn = chatItem.querySelector('.delete-chat-btn');
    if (deleteBtn) {
        deleteBtn.onclick = (e) => {
            e.stopPropagation();
            confirmDeleteChat(chatId, chatItem.querySelector('div').textContent);
        };
    }
    
    chatItem.onclick = () => {
        const chatName = chatItem.querySelector('div').textContent;
        selectChat(chatId, chatName);
    };
}

async function deleteChat(chatId) {
    if (!currentToken || !chatId) {
        console.error('No token or chat ID');
        return;
    }

    try {
        const response = await fetch(`/api/chats/${chatId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });

        if (response.ok) {
            showNotification('Chat deleted successfully');
            
            userChats = userChats.filter(chat => chat.id !== chatId);
            displayChats(userChats);
            
            if (currentChatId === chatId) {
                currentChatId = null;
                document.getElementById('currentChatInfo').textContent = 'Select a chat to start messaging';
                document.getElementById('messages').innerHTML = '';
                document.getElementById('messageInput').placeholder = 'Select a chat to send messages';
                document.getElementById('messageInput').disabled = true;
                document.getElementById('sendButton').disabled = true;
                
                document.querySelectorAll('.chat-item').forEach(item => {
                    item.classList.remove('active');
                });
            }
        } else {
            const data = await response.json();
            throw new Error(data.error || `HTTP ${response.status}`);
        }
    } catch (error) {
        console.error('Failed to delete chat:', error);
        showNotification(`Failed to delete chat: ${error.message}`, 'error');
        
        loadUserChats();
    }
}

function showNotification(message, type = 'info') {
    console.log(`${type.toUpperCase()}: ${message}`);
    
    if (type === 'error') {
        alert(`❌ ${message}`);
    } else {
        alert(`📢 ${message}`);
    }
}

function selectChat(chatId, chatName) {
    currentChatId = chatId;
    
    document.querySelectorAll('.chat-item').forEach(item => {
        item.classList.remove('active');
    });
    event.target.classList.add('active');
    
    document.getElementById('currentChatInfo').textContent = `Chat: ${chatName}`;
    document.getElementById('messageInput').placeholder = `Message in ${chatName}...`;
    document.getElementById('messageInput').disabled = false;
    document.getElementById('sendButton').disabled = false;
    
    joinChat(chatId);
    
    document.getElementById('messages').innerHTML = '';
    loadChatMessages(chatId);
}

async function loadChatMessages(chatId) {
    try {
        const response = await fetch(`/api/chats/${chatId}/messages?limit=50`, {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });

        if (response.ok) {
            const data = await response.json();
            displayMessages(data.messages || []);
        }
    } catch (error) {
        console.error('Failed to load messages:', error);
    }
}

function displayMessages(messages) {
    const messagesDiv = document.getElementById('messages');
    messagesDiv.innerHTML = '';

    if (messages.length === 0) {
        messagesDiv.innerHTML = '<div style="text-align: center; color: #666;">No messages yet</div>';
        return;
    }

    messages.sort((a, b) => new Date(a.timestamp || a.created_at || 0) - new Date(b.timestamp || b.created_at || 0));

    messages.forEach(message => {
        console.log('CUR', currentUsername, 'OLD', message.sender);
        const isOwn = message.sender === currentUsername;
        const sender = message.sender_id || message.sender || 'Unknown';
        const content = message.content || '';
        const timestamp = message.timestamp || message.created_at;
        
        addMessage(sender, content, isOwn ? 'own' : 'other', timestamp);
    });
}

function addMember() {
    const memberInput = document.getElementById('memberInput');
    const member = memberInput.value.trim();
    
    if (member && !currentMembers.includes(member)) {
        currentMembers.push(member);
        updateMembersList();
        memberInput.value = '';
    }
}

function updateMembersList() {
    const membersList = document.getElementById('membersList');
    membersList.innerHTML = currentMembers.map(member => 
        `<span class="member-tag">${member}</span>`
    ).join('');
}

async function testChatAPI() {
    if (!currentToken) {
        console.log('No token available');
        return;
    }

    try {
        console.log('Testing chats API...');
        
        const response = await fetch('/api/chats', {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });
        
        console.log('Chats endpoint status:', response.status);
        console.log('Chats endpoint content-type:', response.headers.get('content-type'));
        
        const text = await response.text();
        console.log('Chats endpoint response (first 200 chars):', text.substring(0, 200));
        
    } catch (error) {
        console.error('Chat API test failed:', error);
    }
    }

async function createChat() {
    const chatName = document.getElementById('chatName').value.trim();
    const messageDiv = document.getElementById('createChatMessage');

    if (!chatName) {
        showMessage(messageDiv, 'Please enter chat name', 'error');
        return;
    }

    if (currentMembers.length === 0) {
        showMessage(messageDiv, 'Please add at least one member', 'error');
        return;
    }

    try {
        const response = await fetch('/api/chats', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${currentToken}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                chat_name: chatName,
                member_ids: currentMembers
            })
        });

        const contentType = response.headers.get('content-type');
        let data;
        
        if (contentType && contentType.includes('application/json')) {
            data = await response.json();
        } else {
            const text = await response.text();
            throw new Error(`Server returned ${response.status}: ${text.substring(0, 100)}`);
        }

        if (response.ok) {
            showMessage(messageDiv, 'Chat created successfully!', 'success');
            
            const membersCopy = [...currentMembers]; 
            
                document.getElementById('chatName').value = '';
            currentMembers = [];
            updateMembersList();
            
            await loadUserChats();
                    
        } else {
            showMessage(messageDiv, data.error || `Failed to create chat: ${response.status}`, 'error');
        }
        } catch (error) {
            console.error('Create chat error:', error);
            showMessage(messageDiv, `Error: ${error.message}`, 'error');
        }
    }

function notifyMembersAboutNewChat(chat, members) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    
    const notification = {
        type: 'chat_created',
        chat_id: chat.id,
        chat_name: chat.name,
        members: members,
        created_by: currentUsername
    };
    
    ws.send(JSON.stringify(notification));
    console.log('Notified members about new chat:', members);
}

function connectWebSocket() {
    console.log('Starting WebSocket connection...');
    
    if (!currentToken) {
        console.error('No token available for WebSocket');
        const savedToken = localStorage.getItem('chatToken');
        const savedUser = localStorage.getItem('chatUsername');
        
        if (savedToken && savedUser) {
            currentToken = savedToken;
            currentUsername = savedUser;
            console.log('Restored from localStorage:', currentUsername);
        } else {
            console.error('No token found in localStorage');
            return;
        }
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${currentToken}`;
    
    console.log('WebSocket URL:', wsUrl);
    
    try {
        ws = new WebSocket(wsUrl);
        
        ws.onopen = function(event) {
            console.log('✅ WebSocket connected successfully!');
            console.log('ReadyState:', ws.readyState);
            
            document.getElementById('messageInput').disabled = false;
            document.getElementById('sendButton').disabled = false;
            
            setTimeout(() => {
                joinAllChats();
            }, 500);
        };
        
        ws.onmessage = function(event) {
            console.log('WebSocket message received:', event.data);
            try {
                const message = JSON.parse(event.data);
                handleWebSocketMessage(message);
            } catch (error) {
                console.log('System message:', event.data);
            }
        };

        
        ws.onclose = function(event) {
            console.log('❌ WebSocket closed:', event.code, event.reason);
            
            document.getElementById('messageInput').disabled = true;
            document.getElementById('sendButton').disabled = true;
            
            setTimeout(() => {
                console.log('Attempting to reconnect...');
                connectWebSocket();
            }, 3000);
        };
        
        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
        };
        
    } catch (error) {
        console.error('Failed to create WebSocket:', error);
    }
}

function joinChat(chatId) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    
    const joinMessage = {
        type: 'join_chat',
        chat_id: parseInt(chatId),
        sender: currentUsername
    };
    ws.send(JSON.stringify(joinMessage));
    console.log('Joined chat:', chatId);
}

function showNotification(message) {
    console.log('Notification:', message);
    alert(`📢 ${message}`);
}

function sendMessage() {
    if (!currentChatId) {
        alert('Please select a chat first');
        return;
    }
    
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        alert('WebSocket not connected. Please refresh the page.');
        return;
    }

    const input = document.getElementById('messageInput');
    const message = input.value.trim();

    if (!message) {
        alert('Message cannot be empty');
        return;
    }

    const numericChatId = parseInt(currentChatId, 10);
    if (isNaN(numericChatId)) {
        alert('Invalid chat ID');
        return;
    }

    console.log('Sending message:', {
        chatId: numericChatId,
        message: message,
        username: currentUsername
    });

    const messageData = {
        type: 'message',
        chat_id: numericChatId,
        content: message,
        sender: currentUsername,
        timestamp: new Date().toISOString()
    };

    try {
        ws.send(JSON.stringify(messageData));
        input.value = '';
        console.log('Message sent successfully');
    } catch (error) {
        console.error('Failed to send message:', error);
        alert('Failed to send message: ' + error.message);
    }
}

function handleWebSocketMessage(message) {
    console.log('Handling WebSocket message:', message);
    
    switch (message.type) {
        case 'message':
            console.log('New message received:', message);
            if (currentChatId && message.chat_id === currentChatId) {
                addMessage(message.sender, message.content, 
                        message.sender === currentUsername ? 'own' : 'other', 
                        message.timestamp);
            } else {
                console.log('Message for different chat or no chat selected');
            }
            break;
            
        case 'chat_created':
            console.log('Chat created notification:', message);
            if (message.members && message.members.includes(currentUsername)) {
                console.log('You were added to new chat:', message.chat_name);
                loadUserChats();
            }
            break;
            
        case 'error':
            console.error('Server error:', message);
            alert('Error: ' + (message.details || message.error));
            break;
            
        case 'joined':
            console.log('Successfully joined chat:', message.chat_id);
            break;
            
        case 'connected':
            console.log('WebSocket connection confirmed:', message.message);
            break;
            
        default:
            console.log('Unknown message type:', message.type, message);
    }
}

function handleKeyPress(event) {
    if (event.key === 'Enter') {
        sendMessage();
    }
}

function showAuth() {
    document.getElementById('authSection').classList.remove('hidden');
    document.getElementById('chatSection').classList.add('hidden');
}

function showChat() {
    document.getElementById('authSection').classList.add('hidden');
    document.getElementById('chatSection').classList.remove('hidden');
    document.getElementById('currentUser').textContent = currentUsername;
    
    updateDebugInfo();
}

function updateDebugInfo() {
    console.log('=== DEBUG INFO ===');
    console.log('Username:', currentUsername);
    console.log('Token present:', !!currentToken);
    console.log('WebSocket state:', ws ? ws.readyState : 'null');
    console.log('Current chat:', currentChatId);
}

function showLoginForm() {
    document.getElementById('loginForm').classList.remove('hidden');
    document.getElementById('registerForm').classList.add('hidden');
    clearMessages();
}

function showRegisterForm() {
    document.getElementById('loginForm').classList.add('hidden');
    document.getElementById('registerForm').classList.remove('hidden');
    clearMessages();
}

function addMessage(sender, text, type, timestamp) {
    const messagesDiv = document.getElementById('messages');
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${type}`;
    
    const senderSpan = document.createElement('div');
    senderSpan.className = 'message-sender';
    senderSpan.textContent = sender;
    
    const textSpan = document.createElement('div');
    textSpan.textContent = text;
    
    messageDiv.appendChild(senderSpan);
    messageDiv.appendChild(textSpan);
    
    // Добавляем timestamp, если есть
    if (timestamp) {
        const timeSpan = document.createElement('div');
        timeSpan.className = 'message-time';
        timeSpan.style.fontSize = '10px';
        timeSpan.style.opacity = '0.7';
        timeSpan.textContent = formatTimestamp(timestamp);
        messageDiv.appendChild(timeSpan);
    }
    
    messagesDiv.appendChild(messageDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

async function loadUserChats() {
    try {
        console.log('Loading user chats...');
        
        const response = await fetch('/api/chats', {
            headers: {
                'Authorization': `Bearer ${currentToken}`
            }
        });

        if (response.ok) {
            const data = await response.json();
            userChats = data.chats || [];
            console.log('Loaded chats:', userChats);
            displayChats(userChats);
            
            // Присоединяемся ко всем чатам через WebSocket
            setTimeout(() => {
                joinAllChats();
            }, 100);
            
        } else {
            console.error('Failed to load chats:', response.status);
            userChats = [];
            displayChats([]);
        }
    } catch (error) {
        console.error('Error loading chats:', error);
        userChats = [];
        displayChats([]);
    }
}

function formatTimestamp(timestamp) {
    try {
        const date = new Date(timestamp);
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch (e) {
        return '';
    }
}

function showMessage(element, message, type) {
    element.textContent = message;
    element.className = type;
    setTimeout(() => element.textContent = '', 5000);
}

function clearMessages() {
    document.getElementById('loginMessage').textContent = '';
    document.getElementById('registerMessage').textContent = '';
}

document.getElementById('messageInput').addEventListener('keypress', function(event) {
    if (event.key === 'Enter') {
        sendMessage();
    }
});

document.addEventListener('DOMContentLoaded', function() {
    const savedToken = localStorage.getItem('chatToken');
    const savedUser = localStorage.getItem('chatUsername');
    
    if (savedToken && savedUser) {
        console.log('Found saved session, auto-login...');
        currentToken = savedToken;
        currentUsername = savedUser;
        
        showChat();
        
        setTimeout(() => {
            loadUserChats();
            connectWebSocket();
        }, 500);
    }
});


document.getElementById('loginUsername').focus();