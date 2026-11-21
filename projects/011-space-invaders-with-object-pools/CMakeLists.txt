cmake_minimum_required(VERSION 3.20)
project(invaders_object_pools LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
  set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
endif()

option(INVADERS_BUILD_GAME "Build the SDL2 game executable" ON)
option(INVADERS_BUILD_TESTS "Build the unit tests" ON)
option(INVADERS_BUILD_BENCH "Build the pool benchmark" ON)

if(MSVC)
  set(INVADERS_WARNINGS /W4 /permissive-)
else()
  set(INVADERS_WARNINGS -Wall -Wextra -Wpedantic -Wconversion -Wshadow)
endif()

# Pure game logic: no SDL, no clock, no I/O.
add_library(invaders_core STATIC
  src/core/Bunker.cpp
  src/core/Config.cpp
  src/core/Game.cpp
)
target_include_directories(invaders_core PUBLIC src)
target_compile_options(invaders_core PRIVATE ${INVADERS_WARNINGS})

if(INVADERS_BUILD_GAME)
  find_package(SDL2 CONFIG QUIET)
  if(SDL2_FOUND)
    add_executable(invaders src/platform/main.cpp src/platform/Renderer.cpp)
    target_link_libraries(invaders PRIVATE invaders_core SDL2::SDL2)
    if(TARGET SDL2::SDL2main)
      target_link_libraries(invaders PRIVATE SDL2::SDL2main)
    endif()
    target_compile_options(invaders PRIVATE ${INVADERS_WARNINGS})
  else()
    message(WARNING "SDL2 not found: building core, tests and benchmark only. "
                    "Install SDL2 (e.g. libsdl2-dev, brew install sdl2, vcpkg install sdl2) to build the game.")
  endif()
endif()

if(INVADERS_BUILD_BENCH)
  add_executable(pool_benchmark bench/pool_benchmark.cpp)
  target_link_libraries(pool_benchmark PRIVATE invaders_core)
  target_compile_options(pool_benchmark PRIVATE ${INVADERS_WARNINGS})
endif()

if(INVADERS_BUILD_TESTS)
  enable_testing()
  find_package(Catch2 3 QUIET)
  if(NOT Catch2_FOUND)
    include(FetchContent)
    FetchContent_Declare(Catch2
      GIT_REPOSITORY https://github.com/catchorg/Catch2.git
      GIT_TAG v3.4.0)
    FetchContent_MakeAvailable(Catch2)
    list(APPEND CMAKE_MODULE_PATH ${catch2_SOURCE_DIR}/extras)
  endif()
  include(Catch)

  add_executable(core_tests
    tests/test_object_pool.cpp
    tests/test_bunker.cpp
    tests/test_config.cpp
    tests/test_game.cpp
  )
  target_link_libraries(core_tests PRIVATE invaders_core Catch2::Catch2WithMain)
  target_compile_options(core_tests PRIVATE ${INVADERS_WARNINGS})
  catch_discover_tests(core_tests)

  # Separate binary: it replaces global operator new to count allocations,
  # which must not interfere with the other tests.
  add_executable(allocation_tests tests/test_no_allocations.cpp)
  target_link_libraries(allocation_tests PRIVATE invaders_core Catch2::Catch2WithMain)
  target_compile_options(allocation_tests PRIVATE ${INVADERS_WARNINGS})
  catch_discover_tests(allocation_tests)
endif()
