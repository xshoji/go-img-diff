# Webサイト向け縦方向DP差分 詳細設計

## 目的

`go-img-diff` の主用途を Web サイトのスクリーンショット比較とし、ページ途中のテキスト増減、画像高さ変化、コンテンツ挿入によって発生する縦方向のずれを吸収したうえで、実差分だけを検出できるようにする。

現状は画像全体に対して 1 回だけ平行移動を推定し、その `DX/DY` を全画素に適用しているため、途中の変化で対応位置がずれると、その下側が全面的に差分扱いになりやすい。

本設計では、

- 横方向は従来通り全体で粗合わせする
- 縦方向は横帯単位で特徴量を取り、動的計画法で再同期する

という二段構成を採用する。

## 対象ユースケース

主対象は以下。

- 同一 viewport 幅で撮影された full-page screenshot
- 上から下へ読む 1 カラムまたは軽い 2 カラムの Web ページ
- 文書ページ、LP、管理画面、EC 商品詳細、ブログ記事、ヘルプページ
- テキスト量や画像高さの変化で、途中以降が縦に押し出されるページ

## 非対象

初期実装では以下を完全には扱わない。

- Masonry レイアウトや Pinterest 型レイアウト
- 左右カラムが独立して大きく伸縮するレイアウト
- 動画、カルーセル、アニメーションが強いページ
- sticky/fixed 要素が画面上に長時間重なるケース
- 大幅な横方向 reflow
- 任意の自然画像比較全般

これらは今後の拡張対象とするが、初期実装の最適化対象には含めない。

## 設計方針

### 基本戦略

1. 既存の pyramid alignment で画像全体の `DX/DY` を粗推定する
2. 画像を一定高さの横帯へ分割する
3. 各帯から Web ページ向けの軽量特徴量を作る
4. 帯同士の類似度行列を作る
5. `match / insert / delete` を持つ DP で最適な帯対応を求める
6. DP の経路を行単位の対応マップへ展開する
7. 差分判定と描画は行単位の対応マップを使って行う

### 期待効果

- 途中のコンテンツ追加や削除があっても、その区間だけを差分として残せる
- その下側は再同期され、全面Diff化を防げる
- 既存の単純平行移動ケースを壊さず拡張できる

## 現行実装との接続点

現在の流れは以下。

1. [internal/app/run.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/app/run.go#L26-L83) で画像読込
2. [internal/align/pyramid.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/align/pyramid.go#L12-L124) で全体アライン
3. [internal/diff/mask.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/diff/mask.go#L12-L62) で画素差分
4. [internal/region/ccl.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/region/ccl.go#L9-L114) で領域化
5. [internal/render/render.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/render/render.go#L12-L75) で描画

変更後は以下にする。

1. 既存の `Align()` はそのまま残し、粗合わせとして利用
2. 新規 `VerticalDPAlign()` を粗合わせ後に実行
3. `BuildMask()` は単一 `DX/DY` ではなく行対応マップを参照
4. `Render()` も行対応マップを参照

## データ構造設計

### 1. 既存 `Alignment`

既存の `core.Alignment` は「全体の粗合わせ結果」として残す。

対象: [internal/core/types.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/core/types.go#L78-L81)

役割:

- 全体 `DX/DY` の推定
- DP の探索範囲を狭めるための基準
- 既存テストとの後方互換維持

### 2. 新規 `RowAlignment`

`core` に以下を追加する。

```go
type RowAlignment struct {
    Width  int
    Height int

    // B の各行 y に対して、A のどの行を参照するか。
    // -1 は A に対応行がないことを表す。
    SrcYByY []int

    // B の各行 y に対して、A 参照時の X オフセット。
    DXByY []int

    // 0..1 の整合度。高いほど良い。
    Score float64
}
```

補助メソッドも追加する。

```go
func NewRowAlignment(width, height, defaultDX, defaultDY int) RowAlignment
func (ra RowAlignment) SrcY(y int) int
func (ra RowAlignment) DX(y int) int
func (ra RowAlignment) HasMapping(y int) bool
```

`NewRowAlignment()` は初期状態として以下を設定する。

- `SrcYByY[y] = y - defaultDY`
- 範囲外は `-1`
- `DXByY[y] = defaultDX`

これにより、DP を無効にした場合でも既存挙動に近い初期値を持てる。

### 3. 新規 `StripeFeature`

`internal/align` 配下に、帯特徴量を表す型を追加する。

```go
type StripeFeature struct {
    Index     int
    StartY    int
    EndY      int
    MeanGray  []float64
    EdgeX     []float64
    InkRatio  float64
    Variance  float64
}
```

意味:

- `MeanGray`: 横方向ビンごとの平均輝度
- `EdgeX`: 横方向の輝度変化量の集計
- `InkRatio`: 文字や図形の占有率に近い値
- `Variance`: 帯全体の強度分散

## 特徴量設計

### 帯分割

初期値:

- `BandHeight = 8`
- `FeatureBins = 32`

理由:

- 高さ 8px は Web テキスト行や余白の変化を追いやすい
- 32 bins は横方向の構造差を表現しつつ、計算量を抑えられる

### 特徴量の詳細

各帯について以下を計算する。

#### MeanGray

- その帯を横方向に `FeatureBins` 分割
- 各 bin の平均輝度を算出
- 値域は `0..255` または `0..1`

期待効果:

- 画像やテキスト塊の横配置を表現できる

#### EdgeX

- 帯内で `abs(gray[x] - gray[x-1])` を集計
- 横方向ビンごとに平均化

期待効果:

- テキストの文字密度や細かい輪郭の違いに強い

#### InkRatio

- 輝度が一定閾値以下、または局所コントラストが高い画素の割合

期待効果:

- 空白帯とコンテンツ帯の識別が容易になる

#### Variance

- 帯内輝度分散

期待効果:

- ベタ画像、空白、複雑テキストの識別補助

### 正規化

特徴量は DP コストでスケール差が暴れないよう正規化する。

方針:

- `MeanGray`, `EdgeX` は bin 数で割って平均距離化
- `InkRatio`, `Variance` は重み付きスカラーとして加算

## 類似度関数

帯 `A[i]` と `B[j]` のマッチコストは以下とする。

```text
cost(i, j) =
    w1 * meanAbsDiff(MeanGrayA, MeanGrayB)
  + w2 * meanAbsDiff(EdgeXA, EdgeXB)
  + w3 * abs(InkRatioA - InkRatioB)
  + w4 * abs(VarianceA - VarianceB)
```

初期重みの候補:

- `w1 = 0.45`
- `w2 = 0.35`
- `w3 = 0.10`
- `w4 = 0.10`

理由:

- Web ページでは輝度分布と文字エッジが最も効きやすい
- `InkRatio`, `Variance` は補助情報として扱う

## DP 設計

### モデル

Needleman-Wunsch 系の sequence alignment を採用する。

状態:

- `dp[i][j]`: `A[0:i]` と `B[0:j]` を比較した最小コスト

遷移:

- `match`: `dp[i-1][j-1] + matchCost(i-1, j-1)`
- `delete`: `dp[i-1][j] + gapPenaltyA`
- `insert`: `dp[i][j-1] + gapPenaltyB`

ここで、

- `delete` は A 側の帯が消えた、または B 側で対応帯が欠落した状態
- `insert` は B 側に新規帯が挿入された状態

### 探索制約

完全な全探索は行数が多いと重くなるため、対角線近傍のみ計算する。

制約:

- `globalBandOffset = round(globalDY / BandHeight)`
- `abs((j - i) - globalBandOffset) <= MaxBandShift`

初期値:

- `MaxBandShift = max(32, imageHeight / BandHeight / 6)`

期待効果:

- 計算量を大きく削減できる
- Web ページの「順序は保たれるが途中でずれる」性質に合う

### Gap penalty

固定値で開始する。

```text
gapPenaltyA = gapPenaltyB = GapPenalty
```

初期値候補:

- `GapPenalty = 18.0`

考え方:

- 小さすぎると不要なギャップが増える
- 大きすぎると誤った無理合わせが増える

初期実装では固定値にし、必要なら後で空白帯や高密度帯に応じて可変にする。

### 空白帯の扱い

空白帯同士のマッチは低コストにする。

具体策:

- `InkRatio` が閾値未満の帯を `blank stripe` とみなす
- blank 同士は `matchCost` をさらに減らす

期待効果:

- 余白の多いページで DP が安定する

### 復元

DP 計算後、backtrace して帯対応列を得る。

出力例:

```text
match(A12, B12)
match(A13, B13)
insert(B14)
insert(B15)
match(A14, B16)
match(A15, B17)
```

この経路を行単位へ展開する。

## 行対応マップへの展開

帯単位の経路から `RowAlignment` を構築する。

### match の場合

`A[i]` と `B[j]` が対応している場合:

- その帯の各行を 1 対 1 で対応付ける
- `SrcYByY[bY] = aY`
- `DXByY[bY] = globalDX`

初期実装では、帯の高さが同じである前提のため単純対応でよい。

### insert の場合

`B[j]` が挿入帯の場合:

- その帯に含まれる全行について `SrcYByY[bY] = -1`
- `DXByY[bY] = globalDX`

差分判定では、`SrcYByY[bY] == -1` を「全面Diff」とみなす。

### delete の場合

`A[i]` のみ存在する場合は、`B` 側には対応帯がないので、直接行は増えない。

ただし復元後の帯オフセットに反映されるため、その後続帯が再同期される。

## 差分判定の変更

対象: [internal/diff/mask.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/diff/mask.go#L12-L62)

### 新しい入力

現在:

```go
func BuildMask(a, b *core.Frame, al core.Alignment, opts core.DiffOptions, logger *slog.Logger) *core.Mask
```

変更後:

```go
func BuildMask(a, b *core.Frame, rowAlign core.RowAlignment, opts core.DiffOptions, logger *slog.Logger) *core.Mask
```

### ロジック

各行 `y` について:

1. `srcY := rowAlign.SrcY(y)` を取得
2. `srcY == -1` なら、その行は全面Diffとして扱う
3. `dx := rowAlign.DX(y)` を取得
4. `ax := x - dx`, `ay := srcY` で A 側画素を参照
5. 既存の閾値判定を適用

### 全面Diffの扱い

挿入行に対しては以下のいずれかを選ぶ。

- 行全体を diff とする
- 画素比較はせず、region 用に行全体を diff マークする

初期実装では単純に行全体 diff でよい。

## 描画の変更

対象: [internal/render/render.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/render/render.go#L12-L75)

### 新しい入力

`Render()` も `RowAlignment` を受け取るよう変更する。

### オーバーレイロジック

現在は `srcX := x - al.DX`, `srcY := y - al.DY` だが、変更後は以下。

1. `srcY := rowAlign.SrcY(y)`
2. `srcY == -1` の場合はオーバーレイしない
3. `srcX := x - rowAlign.DX(y)`
4. 対応画素が存在する場合のみブレンドする

期待効果:

- 挿入帯では無理に A を重ねない
- 再同期後の下側は正しい対応位置からオーバーレイされる

## 新規モジュール構成

### `internal/align/vertical_dp.go`

役割:

- 画像から帯特徴量を構築
- DP による帯対応計算
- `RowAlignment` の生成

主な関数案:

```go
func VerticalDPAlign(a, b *core.Frame, global core.Alignment, opts core.VerticalAlignOptions, logger *slog.Logger) core.RowAlignment
func buildStripeFeatures(f *core.Frame, bandHeight, bins int) []StripeFeature
func stripeMatchCost(a, b StripeFeature, opts core.VerticalAlignOptions) float64
func alignStripesDP(a, b []StripeFeature, globalBandOffset int, opts core.VerticalAlignOptions) dpResult
func expandStripePath(path []dpStep, bHeight, bandHeight, globalDX int) core.RowAlignment
```

### `internal/align/vertical_dp_test.go`

役割:

- 合成データによる経路復元テスト
- 挿入、削除、複数ギャップの回帰テスト

## オプション設計

対象: [internal/core/options.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/core/options.go#L8-L71)

新規に以下を追加する。

```go
type VerticalAlignOptions struct {
    Enabled      bool
    BandHeight   int
    FeatureBins  int
    MaxBandShift int
    GapPenalty   float64
    BlankInkMax  float64
}
```

`Options` に `VerticalAlign VerticalAlignOptions` を追加する。

初期値案:

```go
VerticalAlignOptions{
    Enabled:      true,
    BandHeight:   8,
    FeatureBins:  32,
    MaxBandShift: 0,    // 0 は自動算出
    GapPenalty:   18.0,
    BlankInkMax:  0.03,
}
```

### CLI 対応方針

初回実装では CLI に出さなくてもよい。

理由:

- まず内部挙動として安定化を優先したい
- パラメータ数が増えると利用者負担が大きい

十分安定してから、必要なものだけフラグ公開する。

## 実行フロー変更

対象: [internal/app/run.go](file:///Users/user/Develop/ghq/github.com/xshoji/go-img-diff/internal/app/run.go#L20-L89)

変更後の想定フロー:

1. `frameA`, `frameB` を読み込む
2. `global := align.Align(frameA, frameB, opts.Align, ...)`
3. `rowAlign := align.VerticalDPAlign(frameA, frameB, global, opts.VerticalAlign, logger)`
4. `mask := diff.BuildMask(frameA, frameB, rowAlign, diffOpts, logger)`
5. `regions := region.Extract(mask, ...)`
6. `diffImage := render.Render(frameA, frameB, mask, regions, rowAlign, ...)`

`exit-on-diff` の場合も同じ `rowAlign` を使う。

## 性能設計

### 想定規模

例:

- 画像高さ: 20,000 px
- `BandHeight = 8`
- 帯数: 2,500

完全 DP は重いので、必ず探索バンド制約を入れる。

### 計算量

帯数を `N`, `M`、探索幅を `W` とすると、おおむね `O((N + M) * W)` 相当まで落とせる。

初期実装では以下で十分。

- `MaxBandShift` を制限
- DP テーブル全体ではなく、必要セルのみ計算
- 特徴量計算は 1 回のみ

### メモリ最適化

初版は保守性優先でフルテーブルでもよいが、将来的には以下を検討する。

- コスト計算用は 2 行 rolling buffer
- backtrace 用は別の方向記録テーブル
- もしくはバンド内だけ記録する疎構造

## 品質設計

### テスト項目

最低限必要なテスト:

1. 挿入ブロック 1 箇所
2. 削除ブロック 1 箇所
3. 複数挿入ブロック
4. 単純平行移動のみ
5. 空白帯が多いページ

### 成功条件

以下を主要指標とする。

- 挿入/削除がない tail での diff pixel 数が大きく減る
- 挿入/削除ブロック自体は差分として残る
- 既存の単純オフセットケースで diff が増えない

### 実画像検証

回帰用に Web ページ screenshot の fixture を用意する。

必要ケース:

- テキスト量増加
- 画像高さ増加
- notice bar 追加
- FAQ 展開
- fixed header あり/なし

## 将来拡張

### 1. 区間ごとの `DX` 再推定

初期実装は全体 `DX` 固定だが、match 区間ごとに局所 `DX` を再推定するとさらに強くなる。

適用ケース:

- 一部セクションだけ横にずれている
- scrollbar 有無や軽微な横 reflow

### 2. sticky/fixed 要素の別扱い

画面上部に繰り返し現れる要素は、通常の縦DPでは誤対応の原因になりやすい。

候補:

- 上部数百 px の特徴量重みを下げる
- 同一パターンが複数回出る帯を低信頼扱いにする
- キャプチャ前処理で固定要素を無効化する

### 3. 多カラム向け拡張

必要になった場合、ページ全体を左右分割して別々に縦DPする方式を検討できる。

ただし初期実装の範囲外とする。

## 実装フェーズ

### Phase 1: 基盤追加

- `core.RowAlignment` を追加
- `core.VerticalAlignOptions` を追加
- `BuildMask()` と `Render()` を `RowAlignment` 対応に変更
- 既存 `Alignment` から `RowAlignment` を生成するフォールバックを実装

この段階ではまだ DP を使わない。

### Phase 2: 特徴量と DP 実装

- `buildStripeFeatures()` 実装
- `alignStripesDP()` 実装
- `expandStripePath()` 実装
- `VerticalDPAlign()` を実装

### Phase 3: パイプライン接続

- `Run()` に `VerticalDPAlign()` を組み込む
- `exit-on-diff` でも共通化する
- ログ追加

### Phase 4: 回帰テストと実画像評価

- 合成テスト追加
- 実画像 fixture による評価
- `GapPenalty`, `BandHeight`, `BlankInkMax` の調整

## ログ方針

デバッグしやすいよう、以下を `INFO` または `DEBUG` で出せるようにする。

- 帯数
- `BandHeight`, `FeatureBins`
- `globalDX`, `globalDY`
- `MaxBandShift`
- DP の最終コスト
- `match/insert/delete` 数
- `SrcYByY == -1` の行数

例:

```text
vertical dp complete bandsA=245 bandsB=249 matches=238 inserts=11 deletes=7 score=0.93
```

## 採用判断

本設計の採用基準は以下。

- Web ページ途中の高さ変化で tail 全面Diffになる問題を大幅に減らせる
- 実装コストが光学フローや機械学習より低い
- 既存の単純ケースを壊さず統合できる
- 将来の局所 `DX` 補正や sticky 対策へ自然に拡張できる

このため、本リポジトリの主用途が Web サイト screenshot diff である限り、縦方向DP再同期は最も妥当な中期的アーキテクチャと判断する。
