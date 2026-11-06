# perp README demo. Don't run this directly: use ./demo/record.sh, which
# starts the demo database and prepares the isolated HOME in demo/.tmp.

Output assets/demo.gif

Set Shell "bash"
Set Theme "Catppuccin Mocha"
# Falls back to Fira Code if JetBrains Mono isn't installed (VHS passes this to CSS font-family).
Set FontFamily "JetBrains Mono, Fira Code, monospace"
Set FontSize 16
Set Width 1500
Set Height 720
Set Padding 20
Set TypingSpeed 60ms

# Setup: isolated HOME and a minimal PATH (demo binaries + system dirs only, so
# optional tools like postgres-language-server are not picked up), then launch perp.
Hide
Type `export HOME="$PWD/demo/.tmp/home" XDG_CONFIG_HOME="$PWD/demo/.tmp/home/.config" PATH="$PWD/demo/.tmp/bin:/usr/bin:/bin:/usr/sbin:/sbin" PS1='$ ' HISTFILE=/dev/null`
Enter
Type "clear && perp"
Enter
Wait+Screen@10s /Select a server/
Sleep 500ms
Show

# 1. Server picker
Sleep 1.4s
Enter
Wait+Screen@10s /Connected to server/
Sleep 800ms

# 2. Leader menu -> Database -> List tables
Escape
Space
Sleep 900ms
Type "d"
Sleep 800ms
Type "t"
Wait+Screen@10s /order_items/
Sleep 1.4s

# 3. psql meta-command
Tab
Type "i"
Type "\d books"
Escape
Enter
Wait+Screen@10s /published_on/
Sleep 1.5s

# 4. A real query
Tab
Type "i"
Type "SELECT b.title, a.name AS author, b.genre, b.price FROM books b JOIN authors a ON a.id = b.author_id ORDER BY b.price DESC;"
Sleep 250ms
Escape
Enter
Wait+Screen@10s /author/
Sleep 900ms

# 5. Browse the results
Type "j"
Sleep 250ms
Type "j"
Sleep 250ms
Type "l"
Sleep 250ms
Type "l"
Sleep 400ms

# 6. Export all rows as CSV (leader -> Export -> CSV), then open the export view
Space
Sleep 700ms
Type "e"
Sleep 700ms
Type "c"
Sleep 400ms
Type "priciest-books"
Sleep 300ms
Enter
Wait+Screen@10s /exported/
Sleep 1s
Space
Sleep 500ms
Type "e"
Sleep 500ms
Type "l"
Wait+Screen@10s /priciest-books/
Sleep 2s
