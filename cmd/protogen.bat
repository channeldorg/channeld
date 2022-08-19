cd "%CHANNELD_PATH%\internal\testpb"
protoc --go_out=. --go_opt=paths=source_relative -I . *.proto

cd "%CHANNELD_PATH%\pkg\channeldpb"
protoc --go_out=. --go_opt=paths=source_relative -I . *.proto

cd "%CHANNELD_PATH%\examples\chat-rooms\chatpb"
protoc --go_out=. --go_opt=paths=source_relative -I . *.proto

cd "%CHANNELD_PATH%\examples\channeld-ue-chat\chatpb"
protoc --go_out=. --go_opt=paths=source_relative -I . *.proto

cd "%CHANNELD_PATH%\examples\channeld-ue-tps\tpspb"
protoc --go_out=. --go_opt=paths=source_relative -I . -I ../../.. *.proto

cd "%CHANNELD_PATH%\examples\unity-mirror-tanks\tankspb"
protoc --go_out=. --go_opt=paths=source_relative -I . -I ../../.. *.proto
