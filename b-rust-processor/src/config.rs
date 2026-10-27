//! 설정 읽기. 값은 .env 파일과 환경 변수에서만 읽는다.
//! 같은 이름의 환경 변수가 있으면 환경 변수를 쓴다(b-go-end config.LoadKeys와 같은 규칙).
//! 오류 메시지에는 변수 이름만 넣고 값은 넣지 않는다.

use std::collections::HashMap;
use std::fmt;
use std::io::ErrorKind;
use std::net::SocketAddr;
use std::path::Path;

/// gRPC 서버 주소(IP:포트) 변수. 비어 있으면 DEFAULT_ADDR를 쓴다.
pub const ADDR_VAR: &str = "KEDP_PROCESSOR_ADDR";
pub const DEFAULT_ADDR: &str = "127.0.0.1:50051";
pub const DB_HOST_VAR: &str = "KEDP_DB_HOST";
pub const DB_PORT_VAR: &str = "KEDP_DB_PORT";
pub const DB_NAME_VAR: &str = "KEDP_DB_NAME";
/// DB 계정. Rust는 kedp_rust로만 접속한다(KEDP001-D022 14).
pub const DB_USER: &str = "kedp_rust";
/// kedp_rust 계정의 비밀번호 변수. 이름이 계정 이름과 같다.
pub const DB_PASSWORD_VAR: &str = "kedp_rust";

pub struct Config {
    pub addr: SocketAddr,
    pub db: DbConfig,
}

/// DB 접속 정보. 비밀번호가 들어 있으므로 Debug로 찍어도 비밀번호는 나오지 않게 한다.
#[derive(Clone)]
pub struct DbConfig {
    pub host: String,
    pub port: u16,
    pub name: String,
    pub password: String,
}

impl fmt::Debug for DbConfig {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("DbConfig")
            .field("host", &self.host)
            .field("port", &self.port)
            .field("name", &self.name)
            .field("password", &"***")
            .finish()
    }
}

impl Config {
    /// env_path의 .env 파일과 환경 변수에서 설정을 읽는다. 파일이 없어도 오류가 아니다.
    pub fn load(env_path: &Path) -> Result<Config, String> {
        let file = read_env_file(env_path)?;
        // 환경 변수가 비어 있지 않으면 환경 변수, 아니면 .env 파일 값
        let get = |name: &str| -> Option<String> {
            match std::env::var(name) {
                Ok(v) if !v.is_empty() => Some(v),
                _ => file.get(name).filter(|v| !v.is_empty()).cloned(),
            }
        };
        let require = |name: &str| get(name).ok_or_else(|| format!("{name} is not set"));

        let addr = get(ADDR_VAR)
            .unwrap_or_else(|| DEFAULT_ADDR.to_string())
            .parse()
            .map_err(|_| format!("{ADDR_VAR} is not IP:port"))?;
        let port = require(DB_PORT_VAR)?
            .parse()
            .map_err(|_| format!("{DB_PORT_VAR} is not a port number"))?;
        Ok(Config {
            addr,
            db: DbConfig {
                host: require(DB_HOST_VAR)?,
                port,
                name: require(DB_NAME_VAR)?,
                password: require(DB_PASSWORD_VAR)?,
            },
        })
    }
}

fn read_env_file(path: &Path) -> Result<HashMap<String, String>, String> {
    match std::fs::read_to_string(path) {
        Ok(text) => Ok(parse_env(&text)),
        Err(e) if e.kind() == ErrorKind::NotFound => Ok(HashMap::new()),
        // 오류 메시지에 파일 내용(비밀번호, 키)을 넣지 않는다.
        Err(e) => Err(format!("read env file {}: {}", path.display(), e.kind())),
    }
}

/// NAME=value 줄을 읽는다. 빈 줄과 #으로 시작하는 줄은 건너뛰고, 값을 감싼 따옴표는 벗긴다.
fn parse_env(text: &str) -> HashMap<String, String> {
    let mut env = HashMap::new();
    for line in text.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let Some((name, value)) = line.split_once('=') else {
            continue;
        };
        let mut value = value.trim();
        if value.len() >= 2
            && (value.starts_with('"') || value.starts_with('\''))
            && value.ends_with(&value[..1])
        {
            value = &value[1..value.len() - 1];
        }
        env.insert(name.trim().to_string(), value.to_string());
    }
    env
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_env_reads_names_and_values() {
        let env = parse_env("# 주석\n\nA=1\n B = \"two words\" \nC='3'\nNO_EQUALS\nD=\n");
        assert_eq!(env.get("A").map(String::as_str), Some("1"));
        assert_eq!(env.get("B").map(String::as_str), Some("two words"));
        assert_eq!(env.get("C").map(String::as_str), Some("3"));
        assert_eq!(env.get("D").map(String::as_str), Some(""));
        assert_eq!(env.len(), 4);
    }

    #[test]
    fn debug_hides_password() {
        let db = DbConfig {
            host: "h".into(),
            port: 1,
            name: "n".into(),
            password: "secret-value".into(),
        };
        let shown = format!("{db:?}");
        assert!(!shown.contains("secret-value"), "{shown}");
        assert!(shown.contains("***"));
    }
}
