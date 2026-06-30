pub mod nidus {
    pub mod mesh {
        pub mod v1 {
            tonic::include_proto!("nidus.mesh.v1");
        }
    }
}

pub use nidus::mesh::v1::*;
