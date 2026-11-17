mod common;

use common::*;
use tetris_srs::core::board::{VISIBLE_HEIGHT, WIDTH};
use tetris_srs::core::{
    Board, Config, Event, Game, Input, PieceKind, Rotation, SevenBag, SplitMix64, Status, TopOut,
};

#[test]
fn pieces_spawn_centred_at_the_top() {
    for kind in PieceKind::ALL {
        let game = fixed_game(&[kind], Board::new());
        let p = game.active().unwrap();
        let xs: Vec<i32> = p.cells().iter().map(|c| c.0).collect();
        let (min, max) = (*xs.iter().min().unwrap(), *xs.iter().max().unwrap());
        // Even-width pieces sit dead centre; 3-wide ones round to the left.
        let expected = if kind.box_size() == 3 { 8 } else { 9 };
        assert_eq!(min + max, expected, "{kind:?}");
        let top = p.cells().iter().map(|c| c.1).max().unwrap();
        assert!(top <= VISIBLE_HEIGHT, "{kind:?} top row {top}");
        assert_eq!(p.rotation, Rotation::Zero);
    }
}

#[test]
fn hard_drop_locks_on_the_floor_and_scores_two_per_cell() {
    let mut game = fixed_game(&[PieceKind::O, PieceKind::T], Board::new());
    let start_y = game.active().unwrap().y;
    tap(&mut game, hard_drop());
    assert_eq!(game.board().filled_count(), 4);
    assert!(game.board().get(4, 0).is_some() && game.board().get(5, 1).is_some());
    assert_eq!(game.score(), 2 * start_y as u64);
    assert_eq!(game.active().unwrap().kind, PieceKind::T);
}

#[test]
fn gravity_drops_one_row_per_second_at_level_one() {
    let mut game = fixed_game(&[PieceKind::T], Board::new());
    let y0 = game.active().unwrap().y;
    for _ in 0..59 {
        game.update(DT, &Input::default());
    }
    assert_eq!(game.active().unwrap().y, y0);
    game.update(DT * 1.01, &Input::default());
    assert_eq!(game.active().unwrap().y, y0 - 1);
}

#[test]
fn soft_drop_is_faster_and_scores_one_per_row() {
    let mut game = fixed_game(&[PieceKind::T], Board::new());
    let y0 = game.active().unwrap().y;
    let soft = Input {
        soft_drop: true,
        ..Input::default()
    };
    for _ in 0..30 {
        game.update(DT, &soft);
    }
    let fallen = y0 - game.active().unwrap().y;
    assert_eq!(fallen, 10, "20x gravity for half a second");
    assert_eq!(game.score(), 10);
}

#[test]
fn grounded_piece_waits_for_lock_delay() {
    let mut game = fixed_game(&[PieceKind::T, PieceKind::O], Board::new());
    let soft = Input {
        soft_drop: true,
        ..Input::default()
    };
    while game.active().unwrap().y > -1 {
        game.update(DT, &soft);
    }
    assert_eq!(game.pieces_locked(), 0);
    // 0.5 s lock delay: still active just before, locked just after.
    for _ in 0..28 {
        game.update(DT, &Input::default());
    }
    assert_eq!(game.pieces_locked(), 0);
    assert!(game.lock_progress() > 0.8);
    for _ in 0..4 {
        game.update(DT, &Input::default());
    }
    assert_eq!(game.pieces_locked(), 1);
    assert_eq!(game.active().unwrap().kind, PieceKind::O);
}

#[test]
fn moving_on_the_ground_resets_lock_delay_up_to_the_limit() {
    let config = Config::default();
    let mut game = fixed_game(&[PieceKind::T, PieceKind::O], Board::new());
    let soft = Input {
        soft_drop: true,
        ..Input::default()
    };
    while game.active().unwrap().y > -1 {
        game.update(DT, &soft);
    }
    // Shuffle left and right, each tap after 0.3 s of rest. Without resets
    // the piece would lock during the second wait.
    let mut taps = 0;
    loop {
        for _ in 0..18 {
            game.update(DT, &Input::default());
        }
        if game.pieces_locked() > 0 {
            break;
        }
        let dir = if taps % 2 == 0 { left() } else { right() };
        tap(&mut game, dir);
        taps += 1;
        assert!(taps <= 30, "lock resets never ran out");
    }
    // Taps 1-15 reset the timer; tap 16 doesn't, so the following wait
    // pushes it past 0.5 s.
    assert_eq!(taps, config.max_lock_resets as usize + 1);
}

#[test]
fn reaching_a_new_low_refills_lock_resets() {
    // A step in the floor: the piece burns most of its resets on the ledge,
    // slides off, and gets a fresh set on the lower level.
    let board = Board::from_ascii(&["#####....."]).unwrap();
    let mut game = fixed_game(&[PieceKind::O, PieceKind::T], board);
    let soft = Input {
        soft_drop: true,
        ..Input::default()
    };

    // Park the O on the ledge (x = 3..4, resting on row 0).
    tap(&mut game, left());
    while game.active().unwrap().y > 1 {
        game.update(DT, &soft);
    }
    for _ in 0..6 {
        tap(&mut game, left());
        tap(&mut game, right());
    }
    // 12 resets used. Two more grounded moves carry it off the ledge.
    tap(&mut game, right());
    tap(&mut game, right());
    assert_eq!(game.pieces_locked(), 0);
    while game.active().unwrap().y > 0 {
        game.update(DT, &soft);
    }

    // New lowest row: a full 15 resets are available again.
    for _ in 0..7 {
        tap(&mut game, left());
        tap(&mut game, right());
    }
    assert_eq!(game.pieces_locked(), 0, "14 resets used on the lower level");
    tap(&mut game, left());
    // That was the 15th reset. From here wiggling no longer helps: the
    // 0.5 s timer (30 frames) runs out while the player keeps tapping.
    let mut extra_taps = 0;
    while game.pieces_locked() == 0 {
        let dir = if extra_taps % 2 == 0 { right() } else { left() };
        tap(&mut game, dir);
        extra_taps += 1;
        assert!(extra_taps <= 31, "piece stalled past its lock delay");
    }
    assert!(extra_taps >= 29);
}

#[test]
fn holding_das_slides_to_the_wall() {
    let mut game = fixed_game(&[PieceKind::T], Board::new());
    for _ in 0..40 {
        game.update(DT, &right());
    }
    let max_x = game
        .active()
        .unwrap()
        .cells()
        .iter()
        .map(|c| c.0)
        .max()
        .unwrap();
    assert_eq!(max_x, WIDTH - 1);
}

#[test]
fn hold_swaps_once_per_piece() {
    let mut game = fixed_game(&[PieceKind::T, PieceKind::I, PieceKind::O], Board::new());
    let hold = Input {
        hold: true,
        ..Input::default()
    };
    tap(&mut game, hold);
    assert_eq!(game.held(), Some(PieceKind::T));
    assert_eq!(game.active().unwrap().kind, PieceKind::I);
    assert!(!game.can_hold());
    tap(&mut game, hold);
    assert_eq!(
        game.active().unwrap().kind,
        PieceKind::I,
        "second hold ignored"
    );
    tap(&mut game, hard_drop());
    assert!(game.can_hold());
    tap(&mut game, hold);
    assert_eq!(game.held(), Some(PieceKind::O));
    assert_eq!(game.active().unwrap().kind, PieceKind::T);
}

#[test]
fn preview_shows_upcoming_pieces_in_order() {
    let seq = [
        PieceKind::I,
        PieceKind::O,
        PieceKind::T,
        PieceKind::S,
        PieceKind::Z,
        PieceKind::J,
        PieceKind::L,
    ];
    let mut game = fixed_game(&seq, Board::new());
    let preview: Vec<_> = game.preview().collect();
    assert_eq!(preview, seq[1..6].to_vec());
    tap(&mut game, hard_drop());
    assert_eq!(game.active().unwrap().kind, PieceKind::O);
    assert_eq!(game.preview().next(), Some(PieceKind::T));
}

#[test]
fn vertical_i_into_a_well_scores_a_tetris() {
    let row = "#########.";
    let board = Board::from_ascii(&[row, row, row, row]).unwrap();
    let mut game = fixed_game(&[PieceKind::I, PieceKind::O], board);
    tap(&mut game, cw());
    for _ in 0..40 {
        game.update(DT, &right());
    }
    tap(&mut game, hard_drop());
    let events = game.drain_events();
    let report = events.iter().find_map(|e| match e {
        Event::Scored(r) => Some(*r),
        _ => None,
    });
    let report = report.expect("tetris should score");
    assert_eq!(report.lines, 4);
    assert_eq!(report.points, 800);
    assert!(game.board().is_empty());
    assert_eq!(game.lines(), 4);
}

#[test]
fn level_rises_every_ten_lines() {
    let row = "#########.";
    let board = Board::from_ascii(&[row; 12]).unwrap();
    let mut game = fixed_game(&[PieceKind::I], board);
    let mut level_ups = Vec::new();
    for _ in 0..3 {
        tap(&mut game, cw());
        for _ in 0..40 {
            game.update(DT, &right());
        }
        game.update(0.0, &Input::default());
        tap(&mut game, hard_drop());
        level_ups.extend(game.drain_events().into_iter().filter_map(|e| match e {
            Event::LevelUp(l) => Some(l),
            _ => None,
        }));
    }
    assert_eq!(game.lines(), 12);
    assert_eq!(game.level(), 2);
    assert_eq!(level_ups, vec![2]);
}

#[test]
fn spawning_into_the_stack_is_a_block_out() {
    let mut rows = vec!["####.#####"; 22];
    rows.insert(0, "..........");
    let board = Board::from_ascii(&rows).unwrap();
    let game = fixed_game(&[PieceKind::T], board);
    assert_eq!(game.status(), Status::GameOver(TopOut::BlockOut));
    assert!(game.active().is_none());
}

#[test]
fn locking_above_the_matrix_is_a_lock_out() {
    // Stack reaches row 19 everywhere except where it would clear a line.
    let rows = vec!["#########."; 20];
    let board = Board::from_ascii(&rows).unwrap();
    let mut game = fixed_game(&[PieceKind::O], board);
    // The O spawns in rows 20-21 (no room to drop) over a full column.
    tap(&mut game, left());
    tap(&mut game, hard_drop());
    assert_eq!(game.status(), Status::GameOver(TopOut::LockOut));
}

#[test]
fn invalid_config_is_rejected() {
    let bad = Config {
        lock_delay: -0.1,
        ..Config::default()
    };
    let rng = SplitMix64::new(1);
    assert!(Game::new(bad, Box::new(SevenBag::new(Box::new(rng)))).is_err());
}

#[test]
fn game_over_ignores_further_input() {
    let rows = vec!["####.#####"; 23];
    let board = Board::from_ascii(&rows).unwrap();
    let mut game = fixed_game(&[PieceKind::T], board);
    let before = game.board().clone();
    tap(&mut game, hard_drop());
    assert_eq!(game.board(), &before);
}
