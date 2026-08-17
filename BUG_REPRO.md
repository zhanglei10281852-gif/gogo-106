# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

求一次凸包，输入点的顺序就被改掉了。tessera hull -kind grid -n 25 打出的 input indices 是 0 20 24 4，可这四个顶点分别是 (0,0)、(4,0)、(4,4)、(0,4)，在按行生成的 5x5 网格里它们的下标应该是 0 4 24 20；换成 hull -file examples/pocket.txt 打出的是 0 5 6 4 1，而这七个点里五个凸包顶点的真实下标是 0 2 3 4 5。凸包顶点本身、周长、面积、凸性判定全都是对的，只有回填到输入下标这一步错了，而且错出来的正好是按 x 再按 y 排完序之后的位置。我们的流程是先算凸包拿到边界点的下标，再把同一个点表交给三角化，两边靠下标对齐，现在第一步就把点表顺序换了，后面所有引用都指到了别的点上。mesh 命令因为先三角化再算凸包所以看不出问题。请修复排序会就地改动调用方点表的问题，同时保持排序结果本身按 x 再按 y、去重与凸包结果不变、单点与两点与全共线情形的凸包退化行为不变，并保证全量测试通过。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/gogo-106
- 仓库地址：https://github.com/zhanglei10281852-gif/gogo-106.git
- parent SHA：0e4b74dfcb2274e1bb4203234a2fddb95bf9dc59

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/gogo-106.git bug-repro
cd bug-repro
git checkout --detach 0e4b74dfcb2274e1bb4203234a2fddb95bf9dc59
go test ./internal/geom -run "^TestSortedLexicographicLeavesTheInputAlone$" -count=1 -v
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/geom -run "^TestSortedLexicographicLeavesTheInputAlone$" -count=1 -v
=== RUN   TestSortedLexicographicLeavesTheInputAlone
    sorted_ownership_test.go:18: the input became [(1, 9) (2, 0) (2, 1)], want [(2, 1) (1, 9) (2, 0)]
--- FAIL: TestSortedLexicographicLeavesTheInputAlone (0.00s)
FAIL
FAIL	Tessera/internal/geom	0.002s
FAIL

```

stderr：

```text
warning: internal/geom/sorted_ownership_test.go has type 100755, expected 100644
warning: internal/geom/sorted_ownership_test.go has type 100755, expected 100644

```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/geom -run "^TestSortedLexicographicLeavesTheInputAlone$" -count=1 -v
=== RUN   TestSortedLexicographicLeavesTheInputAlone
    sorted_ownership_test.go:18: the input became [(1, 9) (2, 0) (2, 1)], want [(2, 1) (1, 9) (2, 0)]
--- FAIL: TestSortedLexicographicLeavesTheInputAlone (0.01s)
FAIL
FAIL	Tessera/internal/geom	0.131s
FAIL

```

stderr：

```text
warning: internal/geom/sorted_ownership_test.go has type 100755, expected 100644
warning: internal/geom/sorted_ownership_test.go has type 100755, expected 100644

```

## 通过条件

把一个点表交给排序之后，调用方手里的点表顺序与内容都不变，返回的是按 x 再按 y 排好的独立切片，写返回值不会改到输入；hull -kind grid -n 25 的 input indices 回到 0 4 24 20，hull -file examples/pocket.txt 回到 0 2 3 4 5；凸包顶点、周长、面积、凸性、退化判定与 IndexIn 的对应关系不变；去重保持首次出现顺序、AllCollinear、多边形面积与周长与质心、SortAround 不改动入参、凸裁剪等既有行为不回归；定向测试、全量 go test ./... -count=1 与 go build ./... && go vet ./... 全部通过；校准与远端复跑均在 golang:1.22 linux/amd64 单架构完成。
