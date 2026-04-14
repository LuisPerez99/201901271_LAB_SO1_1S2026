#![allow(missing_docs)]
use kernel::prelude::*;

module! {
    type: ModuloRust,
    name: "modulo_rust",
    authors: ["Luis Pedro Perez Monzon"],
    description: "Un módulo de kernel escrito en Rust que imprime un mensaje al cargar y descargar.",
    license: "GPL",
}

struct ModuloRust;

impl kernel::Module for ModuloRust {
    fn init(_module: &'static kernel::ThisModule) -> Result<Self> {
        pr_info!("¡Hola mundo! Rust está funcionando en el kernel.\n");
        Ok(ModuloRust)
    }
}

impl Drop for ModuloRust {
    fn drop(&mut self) {
        pr_info!("Módulo Rust descargado. ¡Adiós!\n");
    }
}