# Trade-offs e decisões

Esta seção documenta problemas reais encontrados durante a reforma
deste projeto (não hipotéticos): cada um foi reproduzido, corrigido, e
coberto por um teste que falha sem a correção. O código anterior era um
projeto pessoal escrito rapidamente em 2023; nada aqui é crítica a
quem o escreveu sob pressão de tempo, é o registro do que uma segunda
passagem, com mais tempo e testes reais, encontrou.

## 1. O path do módulo usava `/v3` sem nunca ter existido `/v1` ou `/v2`

`go.mod` declarava `module github.com/Jeffersonmf/go-workers/v3`. Pela
convenção de versionamento de módulos Go, um sufixo `/vN` (N >= 2) só é
válido se as tags `v1.x.x`/`v2.x.x` correspondentes já tiverem sido
publicadas. O sufixo declara "esta é a terceira major version", não
"comecei a contar do 3 porque sim". Nenhuma tag jamais existiu neste
repositório. Corrigido removendo o sufixo: `module
github.com/Jeffersonmf/go-workers`.

## 2. `Sugar` podia ser `nil` dependendo da ordem alfabética dos arquivos do pacote

Mantido como registro histórico de um bug real encontrado e corrigido.
A biblioteca envolvida (zap) foi removida depois (item 11); com
`log/slog`, essa classe de bug deixou de ser possível, não só
corrigida.

`pkg/util/logger.go` inicializava o logger `Sugar` dentro de um
`init()`. `pkg/util/config_manager.go` também tinha um `init()` que
loga através de `Sugar` em caso de erro. A especificação de Go garante
a ordem de `init()` *dentro de um arquivo*, e entre arquivos segue a
ordem em que o build os apresenta ao compilador, por convenção, ordem
alfabética do nome do arquivo. `"config_manager.go"` vem antes de
`"logger.go"`, então o `init()` de config rodava primeiro, e `Sugar`
ainda era `nil` quando ele tentava logar, pânico de nil-pointer,
reproduzido ao rodar o binário de exemplo pela primeira vez após a
reforma.

Corrigido movendo a inicialização de `Sugar` para um inicializador de
variável de pacote (`var Sugar = newSugaredLogger()`) em vez de dentro
de um `init()`: a especificação garante que *todo* inicializador de
variável de nível de pacote roda antes de *qualquer* `init()` do mesmo
pacote, independentemente de arquivo: a garantia correta para este
caso, não uma solução que depende de nomear arquivos numa ordem
específica.

## 3. `viper.ConfigFileNotFoundError` era o tipo de erro errado para checar

Mantido como registro histórico. O viper foi removido depois (item
12) em favor de um loader de `.env` sem dependências; esta entrada
documenta um bug real que existiu enquanto o viper ainda estava no
projeto.

O código original (e a primeira tentativa de correção) verificava `err.
(viper.ConfigFileNotFoundError)` para distinguir "`.env` não existe" de
outros erros de leitura. Esse tipo é o que `ReadInConfig` retorna
quando viper *procura* um nome de config em vários diretórios
(`SetConfigName` + `AddConfigPath`) e não encontra em nenhum. Este
projeto usa `SetConfigFile(".env")`, um caminho explícito. Nesse modo,
um arquivo ausente retorna o erro puro do sistema operacional
(`*fs.PathError`), não o tipo do viper. A checagem nunca correspondia
na prática, e todo startup sem `.env` (o caso comum em produção, onde a
config vem de variáveis de ambiente reais) logava um aviso que deveria
ter sido silencioso. Corrigido com `errors.Is(err, fs.ErrNotExist)`.

## 4. `fmt.Sprint(nil)` retorna a string `"<nil>"`, não `""`

Mantido como registro histórico; ver a nota no item 3. O `ReadParameter`
atual (item 12) não passa mais por `viper.Get`.

`ReadParameter` era `fmt.Sprint(viper.Get(parameter))`. Para uma chave
não definida, `viper.Get` retorna `nil` (um `any` nulo), e
`fmt.Sprint(nil)` imprime a string de quatro caracteres `"<nil>"`, não
uma string vazia. Qualquer código chamando
`util.ReadParameter(chave) != ""` para checar se a chave está definida
estava sistematicamente errado para toda chave ausente. Reproduzido por
um teste (`TestReadParameter_UnsetVariableReturnsEmptyString`) que
falhou antes da correção. Resolvido trocando por `viper.GetString`, que
já faz a coerção de tipo correta, incluindo para o caso de chave
ausente.

## 5. O retry original tinha uma race entre uma goroutine e um `select` não bloqueante

Ver a seção correspondente em [`design.md`](./design.md). O padrão era
`go executor(...)` seguido imediatamente por `select { case
<-chPostExecutionFail: retry; default: continue }`. O `select` roda
antes de a goroutine lançada sequer começar, então na prática o
`default` quase sempre vencia e o retry quase nunca disparava de
verdade. `go test -race` sobre a suíte nova (que exercita
`dispatchWithRetry` com uma tarefa que falha duas vezes antes de
suceder) não acusa nenhuma race, porque o redesenho elimina a segunda
goroutine observando o mesmo canal: cada tentativa roda sequencialmente
na goroutine que já está executando aquela execução.

## 6. Nenhum pânico em uma tarefa era de fato recuperado

`WorkerError.ListenErrosHappned` (o nome já tinha um typo) configurava
um `defer`/`recover()`, mas nada no código chamava esse método. Uma
tarefa fornecida pelo chamador que entrasse em pânico derrubava o
processo inteiro. Coberto por
`TestWorker_RecoversFromAPanicInsteadOfCrashing`, que falha (derruba o
processo de teste) sem `safeDispatch` envolvendo a chamada real.

## 7. `ExecutionException`/`WorkerError` declarava uma interface que a própria implementação não satisfazia

```go
type ExecutionException interface {
    RegisterMetricsCount(errorName string, count int64) error
}

func (e *WorkerError) RegisterMetricsCount(
    errorName string, count int64, errorTags []string,
) {
    e.msg = errorTags[0]
}
```

A assinatura da interface tem dois parâmetros e retorna `error`; a
implementação tem três parâmetros e não retorna nada. `*WorkerError`
nunca implementou `ExecutionException`, e nada no código chamava
`RegisterMetricsCount` através da interface de qualquer forma: código
morto que parecia ter um propósito. Substituído por `TaskError`, um
tipo de erro concreto com `Unwrap() error`, e um callback simples
(`Worker.OnError func(*TaskError)`) no lugar da interface.

## 8. `UUIDGenerate` chamava o binário `uuidgen` via `exec.Command`

Funciona em macOS e na maioria das distros Linux com o pacote certo
instalado, mas falha (ou simplesmente não existe) em imagens de
container mínimas, exatamente o tipo de ambiente onde este pacote
rodaria em produção, e paga o custo de criar um processo do sistema
operacional para algo que uma biblioteca em processo já resolve.
`github.com/google/uuid` já era uma dependência indireta (via a cadeia
do viper); tornou-se direta, e `NewUUID()` chama `uuid.NewString()`.

## 9. Substituído `go-co-op/gocron` por `time.Ticker`

O único uso da dependência era `scheduler.Every(N).Seconds().Do(fn)`,
"rodar uma função a cada N segundos", o que `time.Ticker` da biblioteca
padrão já faz. Manter uma dependência (e, por baixo dela,
`robfig/cron/v3` como transitiva) só para esse uso não se paga.
Documentado com mais detalhe em [`design.md`](./design.md#time-ticker-no-lugar-do-go-co-opgocron).

## 10. Histórico do git com o nome de um ex-empregador

Antes desta reforma, o histórico do repositório (não o conteúdo do
`HEAD`, mas diffs de commits antigos) continha o module path
`github.com/tractian/tractian-go-workers` e uma mensagem de log
mencionando a empresa, remanescentes de quando este código era um
projeto interno. O histórico foi comprimido para um único commit antes
de qualquer trabalho novo, removendo essas referências do histórico
publicado por completo, em vez de tentar editar commits antigos um a
um.

## 11. Substituído `zap` por `log/slog` (biblioteca padrão desde Go 1.21)

`Sugar` era um `*zap.SugaredLogger`, construído por `zap.NewProduction()`
dentro de um `init()`. `zap.NewProduction()` pode falhar (retorna um
`error`), o que exigia um caminho de fallback só para esse caso raro,
e foi exatamente a peça em volta da qual o bug do item 2 aconteceu.
`slog.New` nunca falha: não há `error` para tratar nem logger de
fallback para construir, então a classe inteira de bug do item 2 deixa
de ter como acontecer, não só passa a estar corrigida. Quase todo
ponto de log do projeto já usava o estilo chave-valor do zap
(`Infow`/`Warnw`/`Errorw`), que tem a mesma forma dos métodos
`Info`/`Warn`/`Error` do `slog`, então a migração dos call sites foi
mecânica. O nome exportado mudou de `Sugar` para `Logger`, acompanhando
o tipo (`*slog.Logger`, não mais um logger "sugared" com API dupla
printf/chave-valor).

## 12. Substituído `viper` + `fsnotify` por um loader de `.env` sem dependências

O uso real de configuração neste projeto sempre foi: ler `KEY=VALUE`
de um `.env`, com uma variável de ambiente real tendo prioridade. O
viper resolve isso e muito mais (YAML/TOML/HCL/INI, config remota,
watch de diretório), trazendo cerca de uma dezena de dependências
transitivas para uma necessidade de umas vinte linhas de código.
`pkg/util/config_manager.go` agora faz o parse do `.env` diretamente
(`bufio.Scanner` + `strings.Cut`) num `map[string]string` protegido por
`sync.RWMutex`, e `ReadParameter` consulta `os.LookupEnv` primeiro. O
recurso de hot-reload via `fsnotify.WatchConfig` foi removido junto: um
worker de longa duração normalmente é reiniciado pela orquestração
(rolling restart) quando a configuração muda, não recarrega variáveis
de ambiente em memória, e nada no projeto testava esse caminho.
Coberto por `TestLoadEnvFile_PopulatesReadParameterFromDotEnv`,
`TestReadParameter_RealEnvVarOverridesDotEnv` e
`TestLoadEnvFile_MissingFileLeavesNoValuesAndDoesNotPanic`.

Resultado: a árvore de dependências caiu de `viper` + `zap` + `fsnotify`
e cerca de vinte transitivas para uma única dependência direta
(`github.com/google/uuid`) mais `go.uber.org/goleak`, usado só em
teste.
