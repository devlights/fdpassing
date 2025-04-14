# fdpassing

ファイルディスクリプタパッシングのためのGoライブラリです。Unixドメインソケットを利用して、プロセス間でファイルディスクリプタを安全に受け渡しすることができます。

## ファイルディスクリプタパッシングとは

ファイルディスクリプタパッシング（FD passing）とは、あるプロセスから別のプロセスへ、既に開かれたファイルディスクリプタを転送する技術です。Unixシステムでは、ファイル、ソケット、パイプなどのリソースは、プロセス内でファイルディスクリプタという整数値で表現されます。

通常、各プロセスは独自のファイルディスクリプタテーブルを持ちますが、Unixドメインソケットの`SCM_RIGHTS`機能を使用することで、あるプロセスのファイルディスクリプタを別のプロセスに転送し、そのプロセスからも同じリソースにアクセスできるようにすることができます。

**主な利点:**

- リソースの共有: 複数のプロセスが同じファイルやソケットを共有できる
- パフォーマンス向上: 大きなデータをコピーせずに、ファイルディスクリプタだけを転送できる
- 権限の委譲: 特権プロセスが開いたリソースを非特権プロセスに安全に渡すことができる
- サービスの優雅な再起動: サーバープロセスが再起動する際に、接続を失わずに新しいプロセスに転送できる

**技術的な仕組み:**

ファイルディスクリプタパッシングは、Unixドメインソケットの補助データ（ancillary data）機能を使用して実装されます。送信側プロセスは制御メッセージにファイルディスクリプタを含めて送信し、カーネルがそのファイルディスクリプタを受信側プロセスのファイルディスクリプタテーブルに複製します。これにより、受信側プロセスは同じリソースに対する新しいファイルディスクリプタを取得します。

## 概要

`fdpassing`パッケージは、Unixドメインソケットの`SCM_RIGHTS`機能を利用して、あるプロセスから別のプロセスへファイルディスクリプタを転送するためのシンプルなラッパーを提供します。ファイルディスクリプタの転送は、プロセス間でオープンされたファイル、ソケット、パイプなどを共有する場合に特に有用です。

このライブラリは以下の主要な機能を提供します：

- 単一のファイルディスクリプタの送信 (`Send`メソッド)
- 単一のファイルディスクリプタの受信 (`Recv`メソッド)

**注意**: 現在の実装では、一度に1つのファイルディスクリプタのみの送受信をサポートしています。複数のファイルディスクリプタを送信する場合は、Send/Recvメソッドを複数回呼び出す必要があります。

## 使い方

### インストール

```bash
go get github.com/devlights/fdpassing
```

### 基本的な使用例

#### 送信側プロセス

```go
package main

import (
	"log"
	"net"
	"os"
	
	"github.com/devlights/fdpassing"
)

func main() {
	// Unixソケットのパス
	socketPath := "/tmp/fd_socket"
	
	// 既存のソケットを削除
	os.Remove(socketPath)
	
	// リスナーの作成
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		log.Fatalf("リスナーの作成に失敗: %v", err)
	}
	defer listener.Close()
	
	log.Printf("接続待機中: %s", socketPath)
	
	// クライアント接続の受付
	conn, err := listener.AcceptUnix()
	if err != nil {
		log.Fatalf("接続の受付に失敗: %v", err)
	}
	defer conn.Close()
	
	// 送信するファイルを開く
	file, err := os.Open("/path/to/your/file.txt")
	if err != nil {
		log.Fatalf("ファイルのオープンに失敗: %v", err)
	}
	
	// fdpassingのインスタンス作成
	fdSender := fdpassing.NewFd(conn)
	
	// ファイルディスクリプタの送信
	err = fdSender.Send(int(file.Fd()))
	if err != nil {
		log.Fatalf("FDの送信に失敗: %v", err)
	}
	
	log.Println("ファイルディスクリプタを正常に送信しました")
}
```

#### 受信側プロセス

```go
package main

import (
	"io"
	"log"
	"net"
	"os"
	
	"github.com/devlights/fdpassing"
)

func main() {
	// Unixソケットのパス
	socketPath := "/tmp/fd_socket"
	
	// サーバーに接続
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		log.Fatalf("サーバーへの接続に失敗: %v", err)
	}
	defer conn.Close()
	
	// fdpassingのインスタンス作成
	fdReceiver := fdpassing.NewFd(conn)
	
	// ファイルディスクリプタの受信
	receivedFd, err := fdReceiver.Recv()
	if err != nil {
		log.Fatalf("FDの受信に失敗: %v", err)
	}
	
	// 受信したファイルディスクリプタからファイルを作成
	file := os.NewFile(uintptr(receivedFd), "received-file")
	defer file.Close()
	
	// ファイルの内容を読み取り
	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("ファイルの読み取りに失敗: %v", err)
	}
	
	log.Printf("受信したファイルの内容:\n%s", content)
}
```

## どのような場合に有効なライブラリであるか

`fdpassing`ライブラリは以下のようなケースで特に有用です：

1. **マルチプロセスアーキテクチャ**: 親プロセスがオープンしたファイルやソケットを子プロセスに渡す場合

2. **権限の委譲**: 高い権限で実行されているプロセスが、低い権限のプロセスにファイルディスクリプタを渡す場合

3. **ゼロコピー通信**: プロセス間で大量のデータを転送する際に、データをコピーせずにファイルディスクリプタのみを転送することでパフォーマンスを向上させる場合

4. **ホットリロード**: サービスを再起動せずに、新しいプロセスに既存の接続を移管する場合

5. **サンドボックス化されたプロセス**: 制限された環境で実行されるプロセスに、特定のファイルへのアクセスのみを許可する場合

6. **Unixソケットを使った通信**: すでにUnixドメインソケットを使用したIPC（プロセス間通信）を行っているシステムで、ファイルディスクリプタの受け渡しを追加する場合

### 制限事項

- このライブラリはUnixベースのシステム（Linux、macOS、FreeBSDなど）でのみ動作します。Windowsはサポートされていません。
- 送信側と受信側のプロセスは、同一のホストマシン上で実行されている必要があります。ネットワーク越しのファイルディスクリプタパッシングはサポートされていません。
- 現在の実装では、1回の操作で1つのファイルディスクリプタのみを送受信できます。
- 受信したファイルディスクリプタは、使用後に呼び出し側が明示的に閉じる必要があります（`os.NewFile`で作成したファイルの`Close()`メソッドを呼び出す）。
