-- Criar tabela de usuários
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    full_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE
);

-- Criar tabela de produtos
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0,
    category VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Criar tabela de pedidos
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    order_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(10,2) NOT NULL
);

-- Criar tabela de itens do pedido
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id),
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL
);

-- Inserir dados de teste
-- Usuários
INSERT INTO users (username, email, full_name, last_login) VALUES
    ('john_doe', 'john@example.com', 'John Doe', CURRENT_TIMESTAMP - INTERVAL '2 days'),
    ('jane_smith', 'jane@example.com', 'Jane Smith', CURRENT_TIMESTAMP - INTERVAL '1 day'),
    ('bob_wilson', 'bob@example.com', 'Bob Wilson', CURRENT_TIMESTAMP - INTERVAL '3 days');

-- Produtos
INSERT INTO products (name, description, price, stock, category) VALUES
    ('Laptop Pro', 'High-performance laptop with 16GB RAM', 1299.99, 50, 'Electronics'),
    ('Smartphone X', 'Latest model with 128GB storage', 899.99, 100, 'Electronics'),
    ('Wireless Headphones', 'Noise-cancelling headphones', 199.99, 75, 'Accessories'),
    ('Coffee Maker', 'Programmable coffee maker', 79.99, 30, 'Home'),
    ('Fitness Tracker', 'Water-resistant fitness tracker', 149.99, 60, 'Wearables');

-- Pedidos
INSERT INTO orders (user_id, status, total_amount) VALUES
    (1, 'completed', 1499.98),
    (2, 'pending', 899.99),
    (3, 'processing', 279.98);

-- Itens dos pedidos
INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
    (1, 1, 1, 1299.99),
    (1, 3, 1, 199.99),
    (2, 2, 1, 899.99),
    (3, 4, 1, 79.99),
    (3, 5, 1, 149.99);

-- Criar índices para melhor performance
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id); 