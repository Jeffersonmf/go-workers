# Design

## Fluxo

```
Worker.Run()
  -> (com CronSchedulerConfig) runOnSchedule: roda imediatamente, depois a cada intervalo, até ctx cancelar
  -> blockToRun()
       -> sem TickDuration/ExecsPerTick: uma execução, aguardada
       -> com TickDuration/ExecsPerTick: uma tick por elemento, N execuções concorrentes por tick,
          sync.WaitGroup aguarda todas antes de avançar para a próxima
  -> executeTask() por execução:
       -> dispatchWithRetry: até MaxRetries tentativas, sequenciais, na própria goroutine
       -> safeDispatch: recover() em volta do FuncDispatcher do caller
       -> em sucesso, roda NestedCallback; falha (esgotadas as tentativas) vai para OnError
```

## Generics em vez de catorze métodos quase idênticos

`TaskParams` (`pkg/worker_manager/instrumentation.go`) guardava valores
via sete pares de métodos, um por tipo suportado:
`SetStringParam`/`GetStringParam`, `SetIntParam`/`GetIntParam`,
`SetFloatParam`/`GetFloatParam`, `SetBytesParam`/`GetBytesParam`,
`SetRowsParam`/`GetRowsParam`, `SetComplexParam`/`GetComplexParam`, e
mais um par que nem chegava a existir para outros tipos comuns (bool,
slices, structs do próprio chamador). Com Go 1.18+ generics (e a base
de testes deste projeto já em Go 1.27), dois: `SetParam[T](t, key,
value)` e `GetParam[T](t, key) (T, bool)`. Funcionam para qualquer tipo,
incluindo os que o código original nunca previu, sem precisar de mais
um par de métodos a cada novo tipo de dado que um step queira passar
adiante.

Funções livres, não métodos: Go não permite que um método introduza um
parâmetro de tipo além dos do receiver, então `SetParam`/`GetParam`
recebem `TaskParams` como primeiro argumento em vez de serem chamados
como `t.SetParam(...)`, o mesmo padrão de `slices.Contains(s, v)` na
biblioteca padrão.

## O redesenho de concorrência: por que o retry original quase nunca disparava

A implementação anterior lançava uma goroutine por execução e
sinalizava falha por um canal com buffer 1, checado logo em seguida por
um `select` não bloqueante com `default`:

```go
go executor(ctx, taskArg)

select {
case <-chPostExecutionFail:
    // retry
default:
    continue
}
```

Isso roda o `select` imediatamente após lançar a goroutine, sem esperar
ela sequer começar a executar. Na prática, o `select` quase sempre caía
no `default` antes de a goroutine ter tido a chance de enviar no canal,
uma race genuína entre o send e o check, confirmável rodando os testes
novos com `go test -race` (ver `docs/trade-offs.md`).

`dispatchWithRetry` substitui isso por um laço sequencial dentro da
própria goroutine que já está executando aquela tentativa: nada para
disputar, porque não há duas goroutines observando o mesmo estado. Cada
tentativa falha é logada; ao esgotar `MaxRetries`, o erro vai para
`OnError`.

## `safeDispatch`: recuperando pânico que antes não era recuperado

O pacote já tinha uma peça pensada para isso, `ListenErrosHappned`, um
método com `defer`/`recover()`, mas nada no código a chamava. Uma
tarefa do chamador que entrasse em pânico derrubava o processo inteiro,
mesmo que a intenção original fosse claramente proteger contra isso.
`safeDispatch` envolve a chamada real ao `FuncDispatcher` num
`defer`/`recover()` que converte o pânico num `error` normal, tratado
pelo mesmo caminho de retry/`OnError` de qualquer outra falha.

## `sync.WaitGroup`: por que `Run()` agora espera terminar

A implementação anterior lançava as goroutines de uma tick e retornava
sem esperar por elas. Um `main()` que chamasse `Worker.Run()` e saísse
em seguida podia terminar o processo com tarefas ainda no ar, sem opção
de esperar. `runTicks` usa `sync.WaitGroup` para aguardar todas as
execuções de uma tick antes de avançar para a próxima, e `Run()` só
retorna depois que a última tick termina (ou o contexto é cancelado).

## `time.Ticker` no lugar do `go-co-op/gocron`

O uso real da dependência gocron era `scheduler.Every(N).Seconds().Do(fn)`,
exatamente o que `time.Ticker` já faz na biblioteca padrão. Trocar
elimina duas dependências transitivas inteiras (`go-co-op/gocron` e,
por baixo dele, `robfig/cron/v3`) e composição fica mais simples: o
scheduler novo (`runOnSchedule`, em `scheduler.go`) para sozinho quando
o `context.Context` do Worker é cancelado, em vez de precisar de uma
função `Stop` separada e desacoplada do mecanismo de cancelamento que o
resto do pacote já usa.

## `log/slog` no lugar do `zap`, `.env` sem dependências no lugar do `viper`

Depois da primeira reforma (que já tinha corrigido bugs reais dentro do
zap/viper, ver `docs/trade-offs.md` itens 2 a 4), a segunda passagem
substituiu as próprias bibliotecas. `Logger` (`pkg/util/logger.go`) é
um `*slog.Logger` (biblioteca padrão desde Go 1.21) em vez de um
`*zap.SugaredLogger`; `ReadParameter` (`pkg/util/config_manager.go`) lê
`.env` com `bufio.Scanner` e cai para `os.LookupEnv`, sem o viper.

Isso não é só redução de dependências pelo número: `slog.New` não
retorna `error`, então o bug do item 2 (logger `nil` dependendo da
ordem de `init()` entre arquivos) deixa de ter como existir, não só
passa a estar corrigido. Detalhe completo em
[`trade-offs.md`](./trade-offs.md#11-substituído-zap-por-logslog-biblioteca-padrão-desde-go-121)
e
[`trade-offs.md`](./trade-offs.md#12-substituído-viper--fsnotify-por-um-loader-de-env-sem-dependências).

## Encerramento gracioso end to end

O binário de exemplo (`main.go`) usa `signal.NotifyContext` para
transformar `SIGINT`/`SIGTERM` em cancelamento do `context.Context` que
o `Worker` recebe. Testado tanto localmente (`go run .` + `Ctrl+C`)
quanto dentro do container (`docker stop`, que envia `SIGTERM`): em
ambos os casos o processo loga a parada e sai limpo, sem precisar do
`SIGKILL` de última instância do Docker após o período de graça.
