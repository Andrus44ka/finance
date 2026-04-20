// Базовый URL API
const API_URL = '/api';

// Загрузка при старте
document.addEventListener('DOMContentLoaded', () => {
    loadAccounts();
    loadHistory();
});

// ========== РАБОТА СО СЧЕТАМИ ==========

async function loadAccounts() {
    try {
        const response = await fetch(`${API_URL}/accounts`);
        const accounts = await response.json();
        
        // Отображаем счета
        const accountsList = document.getElementById('accounts-list');
        if (accounts.length === 0) {
            accountsList.innerHTML = '<div class="loading">📭 Нет счетов. Создайте первый!</div>';
        } else {
            accountsList.innerHTML = accounts.map(acc => `
                <div class="account-card">
                    <span class="account-name">${escapeHtml(acc.name)}</span>
                    <span class="account-balance">${formatMoney(acc.balance)}</span>
                </div>
            `).join('');
        }
        
        // Обновляем выпадающие списки
        updateSelects(accounts);
        
    } catch (error) {
        showMessage('Ошибка загрузки счетов: ' + error.message, 'error');
    }
}

async function createAccount() {
    const name = document.getElementById('new-account-name').value;
    const balance = parseFloat(document.getElementById('new-account-balance').value);
    
    if (!name) {
        showMessage('Введите название счета', 'error');
        return;
    }
    
    if (isNaN(balance)) {
        showMessage('Введите начальный баланс', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/accounts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, balance })
        });
        
        if (response.ok) {
            showMessage(`✅ Счет "${name}" создан!`, 'success');
            document.getElementById('new-account-name').value = '';
            document.getElementById('new-account-balance').value = '';
            loadAccounts(); // Обновляем список
            loadHistory();  // Обновляем историю
        } else {
            const error = await response.text();
            showMessage('Ошибка: ' + error, 'error');
        }
    } catch (error) {
        showMessage('Ошибка: ' + error.message, 'error');
    }
}

function updateSelects(accounts) {
    const selects = ['income-account', 'expense-account', 'transfer-from', 'transfer-to', 'history-account'];
    
    selects.forEach(selectId => {
        const select = document.getElementById(selectId);
        if (!select) return;
        
        const currentValue = select.value;
        
        if (selectId === 'history-account') {
            select.innerHTML = '<option value="">Все счета</option>';
        } else {
            select.innerHTML = '';
        }
        
        accounts.forEach(acc => {
            const option = document.createElement('option');
            option.value = acc.id;
            option.textContent = `${acc.name} (${formatMoney(acc.balance)})`;
            select.appendChild(option);
        });
        
        // Восстанавливаем выбранное значение, если возможно
        if (currentValue && Array.from(select.options).some(opt => opt.value === currentValue)) {
            select.value = currentValue;
        }
    });
}

// ========== ОПЕРАЦИИ ==========

async function addIncome() {
    const accountId = document.getElementById('income-account').value;
    const amount = parseFloat(document.getElementById('income-amount').value);
    const description = document.getElementById('income-desc').value;
    
    if (!accountId) {
        showMessage('Выберите счет', 'error');
        return;
    }
    
    if (isNaN(amount) || amount <= 0) {
        showMessage('Введите корректную сумму', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/transactions/income`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                account_id: parseInt(accountId),
                amount: amount,
                description: description || 'Пополнение',
                date: new Date().toISOString().split('T')[0]
            })
        });
        
        if (response.ok) {
            showMessage(`✅ Доход ${formatMoney(amount)} добавлен!`, 'success');
            document.getElementById('income-amount').value = '';
            document.getElementById('income-desc').value = '';
            loadAccounts();
            loadHistory();
        } else {
            const error = await response.text();
            showMessage('Ошибка: ' + error, 'error');
        }
    } catch (error) {
        showMessage('Ошибка: ' + error.message, 'error');
    }
}

async function addExpense() {
    const accountId = document.getElementById('expense-account').value;
    const amount = parseFloat(document.getElementById('expense-amount').value);
    const description = document.getElementById('expense-desc').value;
    
    if (!accountId) {
        showMessage('Выберите счет', 'error');
        return;
    }
    
    if (isNaN(amount) || amount <= 0) {
        showMessage('Введите корректную сумму', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/transactions/expense`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                account_id: parseInt(accountId),
                amount: amount,
                description: description || 'Расход',
                date: new Date().toISOString().split('T')[0]
            })
        });
        
        if (response.ok) {
            showMessage(`✅ Расход ${formatMoney(amount)} записан!`, 'success');
            document.getElementById('expense-amount').value = '';
            document.getElementById('expense-desc').value = '';
            loadAccounts();
            loadHistory();
        } else {
            const error = await response.text();
            showMessage('Ошибка: ' + error, 'error');
        }
    } catch (error) {
        showMessage('Ошибка: ' + error.message, 'error');
    }
}

async function addTransfer() {
    const fromId = document.getElementById('transfer-from').value;
    const toId = document.getElementById('transfer-to').value;
    const amount = parseFloat(document.getElementById('transfer-amount').value);
    const description = document.getElementById('transfer-desc').value;
    
    if (!fromId || !toId) {
        showMessage('Выберите счета для перевода', 'error');
        return;
    }
    
    if (fromId === toId) {
        showMessage('Нельзя переводить на тот же счет', 'error');
        return;
    }
    
    if (isNaN(amount) || amount <= 0) {
        showMessage('Введите корректную сумму', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/transactions/transfer`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                from_account_id: parseInt(fromId),
                to_account_id: parseInt(toId),
                amount: amount,
                description: description || 'Перевод',
                date: new Date().toISOString().split('T')[0]
            })
        });
        
        if (response.ok) {
            showMessage(`✅ Перевод ${formatMoney(amount)} выполнен!`, 'success');
            document.getElementById('transfer-amount').value = '';
            document.getElementById('transfer-desc').value = '';
            loadAccounts();
            loadHistory();
        } else {
            const error = await response.text();
            showMessage('Ошибка: ' + error, 'error');
        }
    } catch (error) {
        showMessage('Ошибка: ' + error.message, 'error');
    }
}

// ========== ИСТОРИЯ ==========

async function loadHistory() {
    const accountId = document.getElementById('history-account').value;
    
    let url = `${API_URL}/transactions?limit=50`;
    if (accountId) {
        url += `&account_id=${accountId}`;
    }
    
    try {
        const response = await fetch(url);
        const transactions = await response.json();
        
        const historyList = document.getElementById('history-list');
        
        if (transactions.length === 0) {
            historyList.innerHTML = '<div class="loading">📭 Нет операций</div>';
            return;
        }
        
        historyList.innerHTML = transactions.map(t => {
            let type = 'transfer';
            let amountClass = '';
            let sign = '';
            
            if (t.from_account_id && !t.to_account_id) {
                type = 'expense';
                amountClass = 'negative';
                sign = `-${formatMoney(t.amount)}`;
            } else if (!t.from_account_id && t.to_account_id) {
                type = 'income';
                amountClass = 'positive';
                sign = `+${formatMoney(t.amount)}`;
            } else {
                type = 'transfer';
                amountClass = '';
                sign = `↺ ${formatMoney(t.amount)}`;
            }
            
            const date = new Date(t.date).toLocaleDateString('ru-RU');
            
            return `
                <div class="history-item ${type}">
                    <div class="history-date">${date}</div>
                    <div class="history-desc">${escapeHtml(t.description || 'Без описания')}</div>
                    <div class="history-amount ${amountClass}">${sign}</div>
                </div>
            `;
        }).join('');
        
    } catch (error) {
        console.error('Error loading history:', error);
        document.getElementById('history-list').innerHTML = '<div class="loading">❌ Ошибка загрузки</div>';
    }
}

// ========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ==========

function showTab(tabName) {
    // Скрываем все вкладки
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.classList.remove('active');
    });
    
    // Деактивируем все кнопки
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    
    // Показываем выбранную вкладку
    document.getElementById(`${tabName}-tab`).classList.add('active');
    
    // Активируем кнопку
    event.target.classList.add('active');
}

function formatMoney(amount) {
    return new Intl.NumberFormat('ru-RU', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2
    }).format(amount) + ' ₽';
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showMessage(text, type) {
    const messageDiv = document.getElementById('message');
    messageDiv.textContent = text;
    messageDiv.className = `message ${type}`;
    
    setTimeout(() => {
        messageDiv.className = 'message';
    }, 3000);
}

// Обновляем историю при смене счета
document.addEventListener('DOMContentLoaded', () => {
    const historySelect = document.getElementById('history-account');
    if (historySelect) {
        historySelect.addEventListener('change', loadHistory);
    }
});