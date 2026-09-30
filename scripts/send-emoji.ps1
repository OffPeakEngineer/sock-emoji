param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$Emoji,
    [string]$PipeName = 'sock-emoji',
    [int]$TimeoutMilliseconds = 2000
)

$ErrorActionPreference = 'Stop'
$client = [System.IO.Pipes.NamedPipeClientStream]::new(
    '.', $PipeName, [System.IO.Pipes.PipeDirection]::Out)
try {
    $client.Connect($TimeoutMilliseconds)
    $encoding = [System.Text.UTF8Encoding]::new($false)
    $bytes = $encoding.GetBytes($Emoji + "`n")
    $client.Write($bytes, 0, $bytes.Length)
    $client.Flush()
} finally {
    $client.Dispose()
}
