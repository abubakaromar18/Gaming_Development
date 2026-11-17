mod common;

use common::*;
use tetris_srs::core::tspin::{detect, TSpin};
use tetris_srs::core::{Board, Event, Input, Piece, PieceKind, Rotation};

/// A T-spin double slot: the overhang at column 3 means a T can only get
/// into the slot by dropping in sideways and rotating at the bottom.
///
/// ```text
/// ...#......
/// ###...####
/// ####.#####
/// ```
fn tsd_board() -> Board {
    Board::from_ascii(&["...#......", "###...####", "####.#####"]).unwrap()
}

#[test]
fn three_corner_rule_classifies_full_and_mini() {
    let board = tsd_board();
    let slotted = Piece::new(PieceKind::T, Rotation::Two, 3, 0);
    assert!(board.fits(&slotted));
    assert_eq!(detect(&board, &slotted, Some(0)), TSpin::Full);
    // No rotation as last action: not a spin at all.
    assert_eq!(detect(&board, &slotted, None), TSpin::None);
    // Not a T: never a spin.
    let not_t = Piece {
        kind: PieceKind::J,
        ..slotted
    };
    assert_eq!(detect(&board, &not_t, Some(0)), TSpin::None);
}

#[test]
fn back_corners_only_is_a_mini_unless_fifth_kick() {
    // T pointing up against the floor: both bottom (back) corners are the
    // floor, one front corner is filled, the other is open.
    let board = Board::from_ascii(&["#.........", ".........."]).unwrap();
    let t = Piece::new(PieceKind::T, Rotation::Zero, 0, -1);
    assert!(board.fits(&t));
    assert_eq!(detect(&board, &t, Some(1)), TSpin::Mini);
    assert_eq!(detect(&board, &t, Some(4)), TSpin::Full);
}

#[test]
fn two_corners_is_not_a_spin() {
    let board = Board::new();
    let t = Piece::new(PieceKind::T, Rotation::Zero, 4, -1);
    assert_eq!(detect(&board, &t, Some(0)), TSpin::None);
}

#[test]
fn playing_a_t_spin_double_scores_1200() {
    let mut game = fixed_game(&[PieceKind::T, PieceKind::I], tsd_board());
    // Spawned at x = 3. Turn to R so the stem slides past the overhang.
    tap(&mut game, cw());
    let soft = Input {
        soft_drop: true,
        ..Input::default()
    };
    for _ in 0..200 {
        if game.active().unwrap().y == 0 {
            break;
        }
        game.update(DT, &soft);
    }
    assert_eq!(game.active().unwrap().y, 0, "T should reach the slot");
    tap(&mut game, cw());
    let p = game.active().unwrap();
    assert_eq!((p.rotation, p.x, p.y), (Rotation::Two, 3, 0));
    game.drain_events();

    tap(&mut game, hard_drop());
    let report = game
        .drain_events()
        .into_iter()
        .find_map(|e| match e {
            Event::Scored(r) => Some(r),
            _ => None,
        })
        .expect("the lock should score");
    assert_eq!(report.tspin, TSpin::Full);
    assert_eq!(report.lines, 2);
    assert_eq!(report.points, 1200);
    // The overhang drops to the floor once both rows clear.
    assert_eq!(game.board().filled_count(), 1);
    assert!(game.board().get(3, 0).is_some());
}

#[test]
fn a_blocked_move_after_the_spin_keeps_it() {
    let mut game = fixed_game(&[PieceKind::T, PieceKind::I], tsd_board());
    tap(&mut game, cw());
    let soft = Input {
        soft_drop: true,
        ..Input::default()
    };
    while game.active().unwrap().y > 0 {
        game.update(DT, &soft);
    }
    tap(&mut game, cw());
    // Both sides of the slot are walled in, so these moves fail and the
    // rotation is still the last *successful* action.
    tap(&mut game, right());
    tap(&mut game, left());
    assert_eq!(game.active().unwrap().x, 3);
    tap(&mut game, hard_drop());
    let spun = game
        .drain_events()
        .iter()
        .any(|e| matches!(e, Event::Scored(r) if r.tspin == TSpin::Full));
    assert!(spun);
}

#[test]
fn fifth_kick_upgrades_a_mini_to_full() {
    use tetris_srs::core::srs::{try_rotate, Spin};
    // The T sits on a ledge; tests 1-4 of the 0 -> R rotation all collide,
    // so SRS takes test 5, (-1, -2), and tucks it two rows down under the
    // overhang. Only one front corner is filled, which would be a mini, but
    // the fifth kick always counts as a full T-spin.
    let board = Board::from_ascii(&[
        "..#.......",
        "..........",
        ".#.#......",
        "..........",
        ".#........",
        "..........",
    ])
    .unwrap();
    let t = Piece::new(PieceKind::T, Rotation::Zero, 2, 3);
    assert!(board.fits(&t));
    let r = try_rotate(&board, &t, Spin::Cw).expect("SRS finds a kick");
    assert_eq!(r.kick_index, 4);
    assert_eq!(
        (r.piece.x, r.piece.y, r.piece.rotation),
        (1, 1, Rotation::Right)
    );
    assert_eq!(detect(&board, &r.piece, Some(r.kick_index)), TSpin::Full);
    assert_eq!(detect(&board, &r.piece, Some(1)), TSpin::Mini);
}
