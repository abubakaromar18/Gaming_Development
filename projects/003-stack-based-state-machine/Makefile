# Common tasks. LÖVE runs the folder directly; `make love` packages it.

.PHONY: run test test-all love clean

run:
	love .

test:
	busted

# LÖVE runs LuaJIT; the core also stays portable to PUC Lua 5.1 and 5.4.
test-all:
	busted --lua=luajit
	busted --lua=lua5.1
	busted --lua=lua5.4

love:
	mkdir -p build
	rm -f build/stack-snake.love
	zip -9 -r build/stack-snake.love main.lua conf.lua src

clean:
	rm -rf build
