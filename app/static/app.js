        let ws = null;
        let currentToken = null;
        let currentUsername = null;
        let currentChatId = null;
        let currentMembers = [];
        let userChats = [];
        let currentChatInfo = null;
        let lastMessagesCache = {};

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
                    
                    showMessenger();
                    loadUserChats();
                    
                    setTimeout(() => {
                        connectWebSocket();
                    }, 100);
                } else {
                    if (data.error && data.error.toLowerCase().includes('email not verified')) {
                        showMessage(messageDiv, data.error + ' Would you like to resend the verification email?', 'error');
                        setTimeout(() => {
                            const resendButton = document.createElement('button');
                            resendButton.textContent = 'Resend Verification Email';
                            resendButton.className = 'auth-button auth-switch';
                            resendButton.style.marginTop = '10px';
                            resendButton.onclick = () => resendVerificationForUser(username);
                            messageDiv.appendChild(resendButton);
                        }, 100);
                    } else {
                        showMessage(messageDiv, data.error || 'Login failed', 'error');
                    }
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
                    showVerifyEmail();
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
            document.getElementById('chatMessages').innerHTML = '';
            document.getElementById('contactsScroll').innerHTML = '';
            localStorage.removeItem('chatToken');
            localStorage.removeItem('chatUsername');
        }

        function showVerifyEmail() {
            document.getElementById('authSection').classList.remove('hidden');
            document.getElementById('loginForm').classList.add('hidden');
            document.getElementById('registerForm').classList.add('hidden');
            document.getElementById('verifyEmailSection').classList.remove('hidden');
        }

        async function resendVerification() {
            const email = document.getElementById('registerEmail').value;
            const messageDiv = document.getElementById('registerMessage');

            if (!email) {
                showMessage(messageDiv, 'Email is required', 'error');
                return;
            }

            try {
                const response = await fetch('/api/auth/resend-verification', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ email })
                });

                const data = await response.json();

                if (response.ok) {
                    showMessage(messageDiv, 'Verification email sent!', 'success');
                } else {
                    showMessage(messageDiv, data.error || 'Failed to resend verification email', 'error');
                }
            } catch (error) {
                showMessage(messageDiv, 'Network error: ' + error.message, 'error');
            }
        }

        async function resendVerificationForUser(username) {
            const messageDiv = document.getElementById('loginMessage');
            
            try {
                const response = await fetch('/api/auth/resend-verification', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username })
                });

                const data = await response.json();

                if (response.ok) {
                    showMessage(messageDiv, 'Verification email sent! Please check your inbox.', 'success');
                } else {
                    showMessage(messageDiv, data.error || 'Failed to resend verification email', 'error');
                }
            } catch (error) {
                showMessage(messageDiv, 'Network error: ' + error.message, 'error');
            }
        }

        async function checkVerificationStatus() {
            try {
                const response = await fetch('/api/auth/verification-status', {
                    headers: {
                        'Authorization': `Bearer ${currentToken}`
                    }
                });

                if (response.ok) {
                    const data = await response.json();
                    if (!data.verified) {
                        showNotification('Please verify your email to access all features', 'warning');
                    }
                }
            } catch (error) {
                console.error('Failed to check verification status:', error);
            }
        }

        function showAuth() {
            document.getElementById('authSection').classList.remove('hidden');
            document.getElementById('messengerContainer').classList.add('hidden');
        }

        function showMessenger() {
            document.getElementById('authSection').classList.add('hidden');
            document.getElementById('messengerContainer').classList.remove('hidden');
            document.getElementById('currentUser').textContent = currentUsername;
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

        function showCreateChatModal() {
            document.getElementById('createChatModal').classList.remove('hidden');
        }

        function closeCreateChatModal() {
            document.getElementById('createChatModal').classList.add('hidden');
            document.getElementById('chatName').value = '';
            currentMembers = [];
            updateMembersList();
        }

        function showChatInfo() {
            if (!currentChatInfo) return;
            
            const modal = document.getElementById('chatInfoModal');
            const chatNameElem = document.getElementById('modalChatName');
            const createdElem = document.getElementById('modalChatCreated');
            const membersElem = document.getElementById('modalChatMembers');
            
            chatNameElem.textContent = currentChatInfo.name || 'Unnamed Chat';
            
            const createdDate = new Date(currentChatInfo.created_at);
            createdElem.textContent = createdDate.toLocaleString();
            
            membersElem.innerHTML = '';
            if (currentChatInfo.members && currentChatInfo.members.length > 0) {
                currentChatInfo.members.forEach(member => {
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
            
            modal.classList.remove('hidden');
        }

        function closeChatInfo() {
            document.getElementById('chatInfoModal').classList.add('hidden');
        }

        // Функции чатов
        async function loadUserChats() {
            try {
                const response = await fetch('/api/chats', {
                    headers: {
                        'Authorization': `Bearer ${currentToken}`
                    }
                });

                if (response.ok) {
                    const data = await response.json();
                    userChats = data.chats || [];
                    displayChats(userChats);
                    
                    userChats.forEach(chat => {
                        if (!currentChatInfo && currentChatId === chat.id) {
                            currentChatInfo = chat;
                        }
                    });
                    
                    setTimeout(() => {
                        joinAllChats();
                    }, 100);
                    
                } else {
                    userChats = [];
                    displayChats([]);
                }
            } catch (error) {
                userChats = [];
                displayChats([]);
            }
        }

        function displayChats(chats) {
            const contactsScroll = document.getElementById('contactsScroll');
            contactsScroll.innerHTML = '';

            if (chats.length === 0) {
                contactsScroll.innerHTML = '<div style="color: #666; padding: 20px; text-align: center;">No chats yet</div>';
                return;
            }

            chats.forEach(chat => {
                const contactElement = document.createElement('div');
                contactElement.className = 'contact';
                contactElement.dataset.chatId = chat.id;
                
                const lastMessage = lastMessagesCache[chat.id];
                const displayName = chat.name || `Chat with ${chat.members?.filter(m => m !== currentUsername).join(', ') || 'others'}`;
                
                contactElement.innerHTML = `
                    <div class="contact-avatar">${displayName[0]}</div>
                    <div class="contact-info">
                        <h3>${displayName}</h3>
                        <p>${lastMessage ? `${lastMessage.sender}: ${lastMessage.content.substring(0, 30)}${lastMessage.content.length > 30 ? '...' : ''}` : 'No messages yet'}</p>
                    </div>
                    <button class="delete-chat-btn" onclick="event.stopPropagation(); confirmDeleteChat('${chat.id}', '${displayName.replace(/'/g, "\\'")}')">×</button>
                `;
                
                contactElement.addEventListener('click', () => selectChat(chat.id, displayName, chat));
                contactsScroll.appendChild(contactElement);
                
                if (!lastMessage) {
                    loadLastMessage(chat.id);
                }
            });
        }

        async function loadLastMessage(chatId) {
            try {
                const response = await fetch(`/api/chats/${chatId}/messages?limit=1`, {
                    headers: {
                        'Authorization': `Bearer ${currentToken}`
                    }
                });

                if (response.ok) {
                    const data = await response.json();
                    if (data.messages && data.messages.length > 0) {
                        const lastMessage = data.messages[data.messages.length - 1];
                        lastMessagesCache[chatId] = lastMessage;
                        
                        const contactElement = document.querySelector(`[data-chat-id="${chatId}"]`);
                        if (contactElement) {
                            const messageElem = contactElement.querySelector('p');
                            const shortContent = lastMessage.content.length > 30 
                                ? lastMessage.content.substring(0, 30) + '...' 
                                : lastMessage.content;
                            messageElem.textContent = `${lastMessage.sender}: ${shortContent}`;
                        }
                    }
                }
            } catch (error) {
                console.error('Failed to load last message:', error);
            }
        }

        let stopParticles = null;

function selectChat(chatId, chatName, chatInfo) {
    currentChatId = chatId;
    currentChatInfo = chatInfo;
    
    // Останавливаем предыдущие партиклы
    if (stopParticles) {
        stopParticles();
    }
    
    document.querySelectorAll('.contact').forEach(contact => {
        contact.classList.remove('active');
    });
    event.currentTarget.classList.add('active');
    
    document.getElementById('chatTitle').textContent = chatName;
    document.getElementById('chatTitle').style.cursor = 'pointer';
    document.getElementById('chatTitle').title = 'Click for chat info';
    
    document.getElementById('messageInput').placeholder = `Message in ${chatName}...`;
    document.getElementById('messageInput').disabled = false;
    document.getElementById('sendButton').disabled = false;
    
    joinChat(chatId);
    loadChatMessages(chatId);
    
    // Запускаем новые партиклы
    setTimeout(() => {
        stopParticles = createRightEdgeParticles();
    }, 100);
}

        function resetChatTitle() {
            document.getElementById('chatTitle').textContent = 'Выберите чат';
            document.getElementById('chatTitle').style.cursor = 'default';
            document.getElementById('chatTitle').removeAttribute('title');
        }

        async function loadChatMessages(chatId) {
            try {
                const chatMessages = document.getElementById('chatMessages');
                chatMessages.innerHTML = '<div style="color: #666; padding: 20px; text-align: center;">Loading messages...</div>';
                
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
                const chatMessages = document.getElementById('chatMessages');
                chatMessages.innerHTML = '<div style="color: #dc3545; padding: 20px; text-align: center;">Failed to load messages</div>';
            }
        }

        function displayMessages(messages) {
            const chatMessages = document.getElementById('chatMessages');
            chatMessages.innerHTML = '';

            if (messages.length === 0) {
                chatMessages.innerHTML = '<div style="color: #666; padding: 20px; text-align: center;">No messages yet</div>';
                
                setTimeout(createRightEdgeParticles, 100);
                return;
            }

            messages.sort((a, b) => new Date(a.timestamp || a.created_at || 0) - new Date(b.timestamp || b.created_at || 0));

            messages.forEach(message => {
                const isOwn = message.sender === currentUsername;
                const messageElement = document.createElement('div');
                messageElement.className = `message ${isOwn ? 'my-message' : 'other-message'}`;
                messageElement.innerHTML = `
                    <div class="message-header">
                        <span class="message-sender">${message.sender}</span>
                        <span class="message-time">${formatTimestamp(message.timestamp || message.created_at)}</span>
                    </div>
                    <div class="message-text">${message.content}</div>
                `;
                chatMessages.appendChild(messageElement);
            });

            chatMessages.scrollLeft = chatMessages.scrollWidth - chatMessages.clientWidth;
            initDragToScroll();
            
            setTimeout(createRightEdgeParticles, 100);
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

                const data = await response.json();

                if (response.ok) {
                    showMessage(messageDiv, 'Chat created successfully!', 'success');
                    
                    document.getElementById('chatName').value = '';
                    currentMembers = [];
                    updateMembersList();
                    
                    closeCreateChatModal();
                    await loadUserChats();
                        
                } else {
                    showMessage(messageDiv, data.error || `Failed to create chat: ${response.status}`, 'error');
                }
            } catch (error) {
                showMessage(messageDiv, `Error: ${error.message}`, 'error');
            }
        }

        function confirmDeleteChat(chatId, chatName) {
            const contactElement = document.querySelector(`[data-chat-id="${chatId}"]`);
            const originalContent = contactElement.innerHTML;
            
            contactElement.innerHTML = `
                <div style="text-align: center; width: 100%;">
                    <div style="margin-bottom: 10px; color: #ff9999;">Delete "${chatName}"?</div>
                    <div class="confirm-buttons">
                        <button onclick="deleteChat('${chatId}')" style="background-color: #dc3545;">Yes</button>
                        <button onclick="cancelDelete('${chatId}', \`${originalContent.replace(/`/g, '\\`')}\`)">No</button>
                    </div>
                </div>
            `;
        }

        function cancelDelete(chatId, originalContent) {
            const contactElement = document.querySelector(`[data-chat-id="${chatId}"]`);
            contactElement.innerHTML = originalContent;
            
            const chat = userChats.find(c => c.id === chatId);
            if (chat) {
                const displayName = chat.name || `Chat with ${chat.members?.filter(m => m !== currentUsername).join(', ') || 'others'}`;
                contactElement.addEventListener('click', () => selectChat(chatId, displayName, chat));
            }
        }

        async function deleteChat(chatId) {
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
                        resetChatTitle(); // ← Добавь этот вызов
                        document.getElementById('chatMessages').innerHTML = '';
                        document.getElementById('messageInput').placeholder = 'Выберите чат для отправки сообщений...';
                        document.getElementById('messageInput').disabled = true;
                        document.getElementById('sendButton').disabled = true;
                    }
                } else {
                    throw new Error(`HTTP ${response.status}`);
                }
            } catch (error) {
                showNotification(`Failed to delete chat: ${error.message}`, 'error');
                loadUserChats();
            }
        }

        function connectWebSocket() {
            if (!currentToken) {
                const savedToken = localStorage.getItem('chatToken');
                const savedUser = localStorage.getItem('chatUsername');
                
                if (savedToken && savedUser) {
                    currentToken = savedToken;
                    currentUsername = savedUser;
                } else {
                    return;
                }
            }

            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${currentToken}`;
            
            try {
                ws = new WebSocket(wsUrl);
                
                ws.onopen = function(event) {
                    document.getElementById('messageInput').disabled = false;
                    document.getElementById('sendButton').disabled = false;
                    
                    setTimeout(() => {
                        joinAllChats();
                    }, 500);
                };
                
                ws.onmessage = function(event) {
                    try {
                        const message = JSON.parse(event.data);
                        handleWebSocketMessage(message);
                    } catch (error) {
                        console.log('System message:', event.data);
                    }
                };
                
                ws.onclose = function(event) {
                    document.getElementById('messageInput').disabled = true;
                    document.getElementById('sendButton').disabled = true;
                    
                    setTimeout(() => {
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

        function joinAllChats() {
            if (!ws || ws.readyState !== WebSocket.OPEN) {
                setTimeout(joinAllChats, 1000);
                return;
            }

            userChats.forEach(chat => {
                const joinMessage = {
                    type: 'join_chat',
                    chat_id: chat.id,
                    sender: currentUsername
                };
                ws.send(JSON.stringify(joinMessage));
            });
        }

        function joinChat(chatId) {
            if (!ws || ws.readyState !== WebSocket.OPEN) return;
            
            const joinMessage = {
                type: 'join_chat',
                chat_id: parseInt(chatId),
                sender: currentUsername
            };
            ws.send(JSON.stringify(joinMessage));
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

            const messageData = {
                type: 'message',
                chat_id: numericChatId,
                content: message,
                sender: currentUsername
            };

            try {
                ws.send(JSON.stringify(messageData));
                input.value = '';
            } catch (error) {
                alert('Failed to send message: ' + error.message);
            }
        }

        function handleWebSocketMessage(message) {
            console.log('WebSocket message received:', message);
            
            switch (message.type) {
                case 'message':
                    if (message.chat_id) {
                        lastMessagesCache[message.chat_id] = message;
                        
                        const contactElement = document.querySelector(`[data-chat-id="${message.chat_id}"]`);
                        if (contactElement) {
                            const messageElem = contactElement.querySelector('p');
                            const shortContent = message.content.length > 30 
                                ? message.content.substring(0, 30) + '...' 
                                : message.content;
                            messageElem.textContent = `${message.sender}: ${shortContent}`;
                        }
                    }
                    
                    if (currentChatId && message.chat_id === currentChatId) {
                        const isOwn = message.sender === currentUsername;
                        const messageElement = document.createElement('div');
                        messageElement.className = `message ${isOwn ? 'my-message' : 'other-message'}`;
                        
                        const currentTime = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
                        
                        messageElement.innerHTML = `
                            <div class="message-header">
                                <span class="message-sender">${message.sender}</span>
                                <span class="message-time">${currentTime}</span>
                            </div>
                            <div class="message-text">${message.content}</div>
                        `;
                        document.getElementById('chatMessages').appendChild(messageElement);
                        
                        const chatMessages = document.getElementById('chatMessages');
                        chatMessages.scrollLeft = chatMessages.scrollWidth - chatMessages.clientWidth;
                    }
                    break;
                    
                case 'chat_created':
                    if (message.members && message.members.includes(currentUsername)) {
                        loadUserChats();
                    }
                    break;
                    
                case 'error':
                    alert('Error: ' + (message.details || message.error));
                    break;
            }
        }

        function initDragToScroll() {
            const chatMessages = document.getElementById('chatMessages');
            let isDragging = false;
            let startX = 0, startScrollLeft = 0;

            const startDrag = (e) => {
                isDragging = true;
                startX = (e.pageX !== undefined) ? e.pageX : e.touches[0].pageX;
                startScrollLeft = chatMessages.scrollLeft;
                chatMessages.style.cursor = 'grabbing';
                chatMessages.style.userSelect = 'none';
            };

            const duringDrag = (e) => {
                if (!isDragging) return;
                e.preventDefault();
                const x = (e.pageX !== undefined) ? e.pageX : e.touches[0].pageX;
                const walk = (x - startX);
                chatMessages.scrollLeft = startScrollLeft - walk;
            };

            const stopDrag = () => {
                isDragging = false;
                chatMessages.style.cursor = 'grab';
                chatMessages.style.userSelect = '';
            };

            chatMessages.addEventListener('mousedown', startDrag);
            document.addEventListener('mousemove', duringDrag);
            document.addEventListener('mouseup', stopDrag);

            chatMessages.addEventListener('touchstart', startDrag);
            document.addEventListener('touchmove', duringDrag, { passive: false });
            document.addEventListener('touchend', stopDrag);

            chatMessages.addEventListener('wheel', (e) => {
                e.preventDefault();
                chatMessages.scrollLeft += e.deltaY;
            }, { passive: false });
        }

        function formatTimestamp(timestamp) {
            try {
                if (!timestamp) return '';
                
                const date = new Date(timestamp);
                
                if (isNaN(date.getTime())) {
                    return '';
                }
                
                return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            } catch (e) {
                console.error('Error formatting timestamp:', e, timestamp);
                return '';
            }
        }

        function showMessage(element, message, type) {
            element.textContent = message;
            element.className = type;
            setTimeout(() => element.textContent = '', 5000);
        }

        function showNotification(message, type = 'info') {
            if (type === 'error') {
                alert(`❌ ${message}`);
            } else {
                alert(`📢 ${message}`);
            }
        }

        function clearMessages() {
            document.getElementById('loginMessage').textContent = '';
            document.getElementById('registerMessage').textContent = '';
        }

        function createRightEdgeParticles() {
            const chatMessages = document.getElementById('chatMessages');
            
            const oldParticles = document.querySelector('.chat-particles-container');
            if (oldParticles) oldParticles.remove();
            
            const particlesContainer = document.createElement('div');
            particlesContainer.className = 'chat-particles-container';
            document.querySelector('.chat-messages-container').appendChild(particlesContainer);
            
            function createParticleWave() {
                const particleCount = 8 + Math.floor(Math.random() * 8);
                
                for (let i = 0; i < particleCount; i++) {
                    const particle = document.createElement('div');
                    
                    if (Math.random() > 0.7) {
                        particle.className = 'particle speed-line';
                    } else {
                        particle.className = 'particle';
                    }
                    
                    const startY = Math.random() * 100;
                    const moveX = -50 - Math.random() * 100;
                    const moveY = -20 + Math.random() * 40;
                    
                    const delay = Math.random() * 2;
                    const duration = 2 + Math.random() * 2;
                    
                    particle.style.setProperty('--start-y', `${startY}%`);
                    particle.style.setProperty('--move-x', `${moveX}px`);
                    particle.style.setProperty('--move-y', `${moveY}px`);
                    particle.style.animationDelay = `${delay}s`;
                    particle.style.animationDuration = `${duration}s`;
                    
                    particlesContainer.appendChild(particle);
                    
                    setTimeout(() => {
                        if (particle.parentNode === particlesContainer) {
                            particle.remove();
                        }
                    }, (delay + duration) * 1000);
                }
            }
            
            const particleInterval = setInterval(createParticleWave, 1500);
            
            createParticleWave();
            
            return () => clearInterval(particleInterval);
        }

        function hideSplashScreen() {
            const splashScreen = document.getElementById('splashScreen');
            splashScreen.classList.add('hidden');
            
            setTimeout(() => {
                splashScreen.style.display = 'none';
            }, 800);
        }

        document.getElementById('messageInput').addEventListener('keypress', function(event) {
            if (event.key === 'Enter') {
                sendMessage();
            }
        });

        document.addEventListener('DOMContentLoaded', function() {
            const urlParams = new URLSearchParams(window.location.search);
            const verificationToken = urlParams.get('token');
            
            if (verificationToken) {
                handleEmailVerification(verificationToken);
            }
            
            setTimeout(() => {
                hideSplashScreen();
                
                const savedToken = localStorage.getItem('chatToken');
                const savedUser = localStorage.getItem('chatUsername');
                
                if (savedToken && savedUser) {
                    currentToken = savedToken;
                    currentUsername = savedUser;
                    
                    setTimeout(() => {
                        showMessenger();
                        
                        setTimeout(() => {
                            loadUserChats();
                            connectWebSocket();
                            checkVerificationStatus(); // Проверяем статус верификации
                            setTimeout(createRightEdgeParticles, 500);
                        }, 500);
                    }, 300);
                } else {
                    setTimeout(() => {
                        showAuth();
                        document.getElementById('loginUsername').focus();
                    }, 300);
                }
            }, 3000);
            });

            async function handleEmailVerification(token) {
                try {
                    const response = await fetch(`/api/auth/verify-email?token=${token}`);
                    const data = await response.json();

                    if (response.ok) {
                        showNotification('Email verified successfully! You can now login.', 'success');
                        window.history.replaceState({}, document.title, window.location.pathname);
                    } else {
                        showNotification(data.error || 'Email verification failed', 'error');
                    }
                } catch (error) {
                    showNotification('Verification failed: ' + error.message, 'error');
                }
            }

        document.getElementById('sendButton').addEventListener('click', sendMessage);