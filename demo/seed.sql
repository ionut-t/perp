-- Fictional bookshop used for the perp demo. All names are made up.
SELECT setseed(0.42);

CREATE TABLE authors (
    id         serial PRIMARY KEY,
    name       text NOT NULL,
    country    text NOT NULL,
    born       int
);

CREATE TABLE books (
    id           serial PRIMARY KEY,
    author_id    int NOT NULL REFERENCES authors(id),
    title        text NOT NULL,
    genre        text NOT NULL,
    price        numeric(6,2) NOT NULL,
    published_on date NOT NULL,
    stock        int NOT NULL DEFAULT 0
);

CREATE TABLE customers (
    id         serial PRIMARY KEY,
    name       text NOT NULL,
    email      text NOT NULL UNIQUE,
    city       text NOT NULL,
    joined_at  timestamptz NOT NULL
);

CREATE TABLE orders (
    id           serial PRIMARY KEY,
    customer_id  int NOT NULL REFERENCES customers(id),
    status       text NOT NULL CHECK (status IN ('pending', 'shipped', 'delivered', 'cancelled')),
    ordered_at   timestamptz NOT NULL
);

CREATE TABLE order_items (
    order_id  int NOT NULL REFERENCES orders(id),
    book_id   int NOT NULL REFERENCES books(id),
    quantity  int NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (order_id, book_id)
);

CREATE INDEX books_author_id_idx ON books(author_id);
CREATE INDEX orders_customer_id_idx ON orders(customer_id);

-- 30 authors
INSERT INTO authors (name, country, born)
SELECT f || ' ' || l,
       (ARRAY['UK','Ireland','Canada','Norway','Chile','Japan','Italy','Kenya','Poland','Portugal'])[1 + (i % 10)],
       1940 + (i * 7) % 50
FROM generate_series(1, 30) AS i,
     LATERAL (SELECT (ARRAY['Ada','Bram','Cleo','Dmitri','Esme','Felix','Greta','Hugo','Iris','Jonas',
                            'Kira','Leon','Mira','Nico','Odile'])[1 + (i % 15)] AS f,
                     (ARRAY['Quill','Inkwell','Marlow','Fenwick','Hollow','Ashby','Penrose','Larkspur',
                            'Thistle','Vane','Wren','Calloway','Brightwater'])[1 + ((i * 5) % 13)] AS l) n;

-- 120 books
INSERT INTO books (author_id, title, genre, price, published_on, stock)
SELECT 1 + (i % 30),
       'The ' || a || ' ' || b,
       (ARRAY['Fantasy','Mystery','Sci-Fi','Romance','History','Poetry','Thriller'])[1 + (i % 7)],
       round((6 + random() * 24)::numeric, 2),
       date '1985-01-01' + (random() * 14000)::int,
       (random() * 60)::int
FROM generate_series(1, 120) AS i,
     LATERAL (SELECT (ARRAY['Silent','Crimson','Forgotten','Hollow','Last','Velvet','Paper','Midnight',
                            'Glass','Winter','Copper','Wandering'])[1 + (i % 12)] AS a,
                     (ARRAY['Lighthouse','Orchard','Cartographer','Archive','Tide','Garden','Comet',
                            'Harbour','Clockmaker','Lantern','Meridian'])[1 + ((i * 7) % 11)] AS b) t;

-- 80 customers
INSERT INTO customers (name, email, city, joined_at)
SELECT f || ' ' || l,
       lower(f) || '.' || lower(l) || i || '@example.com',
       (ARRAY['Bristol','Galway','Halifax','Bergen','Valparaiso','Osaka','Turin','Mombasa','Krakow','Porto'])[1 + (i % 10)],
       timestamptz '2022-01-01' + (random() * 900) * interval '1 day'
FROM generate_series(1, 80) AS i,
     LATERAL (SELECT (ARRAY['Alex','Sam','Robin','Jamie','Taylor','Morgan','Casey','Riley','Quinn','Avery',
                            'Jordan'])[1 + (i % 11)] AS f,
                     (ARRAY['Example','Sample','Testwood','Placeholder','Mockford','Demo','Fakeley'])[1 + ((i * 3) % 7)] AS l) n;

-- 300 orders
INSERT INTO orders (customer_id, status, ordered_at)
SELECT 1 + (random() * 79)::int,
       (ARRAY['delivered','delivered','delivered','shipped','pending','cancelled'])[1 + (random() * 5)::int],
       timestamptz '2024-01-01' + (random() * 670) * interval '1 day'
FROM generate_series(1, 300);

-- 1–4 line items per order
INSERT INTO order_items (order_id, book_id, quantity)
SELECT o.id, b.book_id, 1 + (random() * 2)::int
FROM orders o,
     LATERAL (SELECT DISTINCT 1 + ((o.id * 37 + k * 53) % 120) AS book_id
              FROM generate_series(1, 1 + (o.id % 4)) AS k) b;

ANALYZE;
