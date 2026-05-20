(() => {
  'use strict';

  let currentConvId = null;
  let convs = [];
  let loading = false;

  const el = (id) => document.getElementById(id);
  const msgList = el('messages');
  const convList = el('conv-list');
  const msgInput = el('msg-input');
  const btnSend = el('btn-send');
  const btnImage = el('btn-image');
  const btnNewChat = el('btn-new-chat');
  const fileInput = el('file-input');
  const chatTitle = el('chat-title');
  const btnMenu = el('btn-menu');
  const sidebar = el('sidebar');

  // --- API ---
  async function api(path, opts = {}) {
    const res = await fetch(path, {
      headers: { 'Content-Type': 'application/json', ...opts.headers },
      ...opts,
    });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    if (res.status === 204) return null;
    return res.json();
  }

  // --- Conversations ---
  async function loadConvs() {
    convs = await api('/api/conversations');
    renderConvs();
  }

  function renderConvs() {
    convList.innerHTML = '';
    convs.forEach(c => {
      const div = document.createElement('div');
      div.className = 'conv-item' + (c.id === currentConvId ? ' active' : '');
      div.innerHTML = `<div class="conv-title">${esc(c.title || 'New Chat')}</div>
        <div class="conv-time">${timeAgo(c.updated_at)}</div>`;
      div.onclick = () => selectConv(c.id);
      convList.appendChild(div);
    });
  }

  async function selectConv(id) {
    if (id === currentConvId) return;
    currentConvId = id;
    renderConvs();
    const data = await api(`/api/conversations/${id}`);
    renderMessages(data.messages);
    chatTitle.textContent = data.conversation.title || 'Chat';
    msgInput.focus();
    if (window.innerWidth <= 768) sidebar.classList.remove('open');
  }

  async function newConv() {
    currentConvId = null;
    msgList.innerHTML = '';
    chatTitle.textContent = 'melon-chat';
    loadConvs();
    msgInput.focus();
  }

  // --- Messages ---
  function renderMessages(msgs) {
    msgList.innerHTML = '';
    msgs.forEach(m => appendMessage(m));
    scrollToBottom();
  }

  function appendMessage(m) {
    const div = document.createElement('div');
    div.className = `msg ${m.role}${m.content_type === 'recognition_result' ? ' recognition_result' : ''}`;
    div.id = `msg-${m.id}`;

    let content = '';
    if (m.image_url) {
      content += `<img class="msg-image" src="${m.image_url}" alt="image" loading="lazy">`;
    }
    if (m.content) {
      content += `<div class="msg-content">${esc(m.content).replace(/\n/g, '<br>')}</div>`;
    }
    content += `<div class="msg-time">${timeStr(m.created_at)}</div>`;
    div.innerHTML = content;
    msgList.appendChild(div);

    div.querySelectorAll('.msg-image').forEach(img => {
      img.onclick = () => showPreview(img.src);
    });

    scrollToBottom();
  }

  async function sendMessage() {
    const text = msgInput.value.trim();
    if (!text && !pendingImage) return;
    if (loading) return;

    loading = true;
    btnSend.disabled = true;

    const body = { content_type: 'text', content: text };
    if (currentConvId) body.conversation_id = currentConvId;

    if (pendingImage) {
      body.content_type = 'image';
      body.image = pendingImage;
      pendingImage = null;
      updateImageBtn();
    }

    try {
      const result = await api('/api/chat', {
        method: 'POST',
        body: JSON.stringify(body),
      });
      if (!currentConvId) {
        currentConvId = result.conversation_id;
        chatTitle.textContent = text.slice(0, 50) || 'Chat';
        renderConvs();
      }
      msgInput.value = '';
    } catch (err) {
      appendSystemMsg('送信エラー: ' + err.message);
    } finally {
      loading = false;
      btnSend.disabled = false;
      msgInput.focus();
    }
  }

  function appendSystemMsg(text) {
    const div = document.createElement('div');
    div.className = 'msg system';
    div.textContent = text;
    msgList.appendChild(div);
    scrollToBottom();
  }

  // --- Image ---
  let pendingImage = null;

  function updateImageBtn() {
    btnImage.textContent = pendingImage ? '🖼️' : '📷';
  }

  btnImage.onclick = () => fileInput.click();

  fileInput.onchange = () => {
    const file = fileInput.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (e) => {
      pendingImage = e.target.result;
      updateImageBtn();
      msgInput.focus();
      if (!msgInput.value.trim()) {
        sendMessage();
      }
    };
    reader.readAsDataURL(file);
    fileInput.value = '';
  };

  // --- Preview ---
  function showPreview(src) {
    const overlay = document.createElement('div');
    overlay.id = 'image-preview';
    overlay.style.display = 'flex';
    overlay.innerHTML = `<img src="${src}" alt="preview">`;
    overlay.onclick = () => overlay.remove();
    document.body.appendChild(overlay);
  }

  // --- SSE ---
  function connectSSE() {
    const es = new EventSource('/api/events');
    es.addEventListener('message', (e) => {
      try {
        const data = JSON.parse(e.data);
        if (data.type === 'message') {
          const msg = data.payload;
          if (msg.conversation_id === currentConvId) {
            appendMessage(msg);
          }
          loadConvs();
        }
      } catch (err) {
        console.warn('SSE parse:', err);
      }
    });
    es.addEventListener('connected', () => {
      el('status-indicator').style.background = '#4caf50';
    });
    es.onerror = () => {
      el('status-indicator').style.background = '#e94560';
    };
    return es;
  }

  // --- Helpers ---
  function esc(s) {
    const d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
  }

  function timeStr(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    return d.toLocaleTimeString('ja-JP', { hour: '2-digit', minute: '2-digit' });
  }

  function timeAgo(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    const now = Date.now();
    const diff = now - d.getTime();
    if (diff < 60000) return 'just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h`;
    return d.toLocaleDateString('ja-JP');
  }

  function scrollToBottom() {
    requestAnimationFrame(() => {
      msgList.scrollTop = msgList.scrollHeight;
    });
  }

  // --- Events ---
  btnSend.onclick = sendMessage;
  msgInput.onkeydown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  btnNewChat.onclick = newConv;
  btnMenu.onclick = () => sidebar.classList.toggle('open');

  // --- Init ---
  async function init() {
    await loadConvs();
    if (convs.length > 0) {
      await selectConv(convs[0].id);
    }
    connectSSE();

    document.addEventListener('click', (e) => {
      if (window.innerWidth <= 768 && sidebar.classList.contains('open')) {
        if (!sidebar.contains(e.target) && e.target !== btnMenu) {
          sidebar.classList.remove('open');
        }
      }
    });
  }

  init();
})();
