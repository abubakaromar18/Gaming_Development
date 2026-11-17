//! Pure game logic. Nothing in here knows about windows, keys, sound or the
//! wall clock: time arrives as `dt`, randomness as an injected source.

pub mod autorepeat;
pub mod board;
pub mod config;
pub mod game;
pub mod piece;
pub mod randomizer;
pub mod rng;
pub mod scoring;
pub mod srs;
pub mod tspin;

pub use board::Board;
pub use config::Config;
pub use game::{Event, Game, Input, Status, TopOut};
pub use piece::{Piece, PieceKind, Rotation};
pub use randomizer::{FixedSequence, Randomizer, SevenBag};
pub use rng::{RandomSource, SplitMix64};
pub use scoring::ClearReport;
pub use tspin::TSpin;
