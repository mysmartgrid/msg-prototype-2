project(lua-ws-wrapper)
cmake_minimum_required(VERSION "2.8" FATAL_ERROR)

set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -std=c99")
add_executable(lua-ws-wrapper "${CMAKE_CURRENT_SOURCE_DIR}/ws.c")
target_link_libraries(lua-ws-wrapper websockets ssl crypto)
