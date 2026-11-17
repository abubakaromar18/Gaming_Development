//! Long randomized sessions: the game must never panic, never let the
//! active piece overlap the stack, conserve cells, and be bit-for-bit
//! reproducible from its seeds.

mod common;

use common::DT;
use tetris_srs::core::{Board, Config, Game, Input, RandomSource, SevenBag, SplitMix64, Status};

struct Summary {
    score: u64,
    lines: u32,
    pieces: u64,
    board: Board,
    frames: u32,
}

fn random_input(rng: &mut SplitMix64) -> Input {
    let bits = rng.next_u64();
    let bit = |n: u32, odds: u64| (bits >> (n * 8)) & 0xFF < odds;
    Input {
        left: bit(0, 60),
        right: bit(1, 60),
        soft_drop: bit(2, 40),
        hard_drop: bit(3, 6),
        rotate_cw: bit(4, 20),
        rotate_ccw: bit(5, 20),
        hold: bit(6, 4),
    }
}

fn play(piece_seed: u64, input_seed: u64, max_frames: u32) -> Summary {
    let bag = SevenBag::new(Box::new(SplitMix64::new(piece_seed)));
    let mut game = Game::new(Config::default(), Box::new(bag)).unwrap();
    let mut inputs = SplitMix64::new(input_seed);
    let mut frames = 0;
    while frames < max_frames && game.status() == Status::Playing {
        game.update(DT, &random_input(&mut inputs));
        game.drain_events();
        frames += 1;
        if let Some(p) = game.active() {
            assert!(
                game.board().fits(p),
                "active piece overlaps at frame {frames}"
            );
        }
        // Every locked piece adds four cells; every cleared line removes ten.
        let expected = 4 * game.pieces_locked() as usize - 10 * game.lines() as usize;
        assert_eq!(
            game.board().filled_count(),
            expected,
            "cells lost at frame {frames}"
        );
    }
    Summary {
        score: game.score(),
        lines: game.lines(),
        pieces: game.pieces_locked(),
        board: game.board().clone(),
        frames,
    }
}

#[test]
fn many_random_games_keep_invariants() {
    let mut total_frames = 0;
    let mut total_pieces = 0;
    // Random mashing tops out within a few seconds, so play many games.
    for seed in 0..200 {
        let s = play(seed, seed ^ 0xDEAD_BEEF, 20_000);
        total_frames += s.frames;
        total_pieces += s.pieces;
    }
    assert!(total_frames > 100_000);
    assert!(total_pieces > 2_000);
}

#[test]
fn same_seeds_same_game() {
    let a = play(5, 6, 30_000);
    let b = play(5, 6, 30_000);
    assert_eq!(
        (a.score, a.lines, a.pieces, a.frames),
        (b.score, b.lines, b.pieces, b.frames)
    );
    assert_eq!(a.board, b.board);
    let c = play(5, 7, 30_000);
    assert!(a.board != c.board || a.score != c.score);
}
