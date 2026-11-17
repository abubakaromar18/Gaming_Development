use std::collections::HashSet;

use tetris_srs::core::{PieceKind, Randomizer, SevenBag, SplitMix64};

fn deal(seed: u64, n: usize) -> Vec<PieceKind> {
    let mut bag = SevenBag::new(Box::new(SplitMix64::new(seed)));
    (0..n).map(|_| bag.next_piece()).collect()
}

#[test]
fn every_bag_is_a_permutation_of_all_seven() {
    let pieces = deal(42, 7 * 1000);
    for chunk in pieces.chunks(7) {
        let set: HashSet<_> = chunk.iter().collect();
        assert_eq!(set.len(), 7, "bag {chunk:?}");
    }
}

#[test]
fn droughts_never_exceed_twelve() {
    let pieces = deal(7, 70_000);
    for kind in PieceKind::ALL {
        let mut gap = 0;
        for &p in &pieces {
            if p == kind {
                gap = 0;
            } else {
                gap += 1;
                assert!(gap <= 12, "{kind:?} drought of {gap}");
            }
        }
    }
}

#[test]
fn same_seed_same_sequence() {
    assert_eq!(deal(123, 500), deal(123, 500));
    assert_ne!(deal(123, 500), deal(124, 500));
}

#[test]
fn bag_order_is_roughly_uniform() {
    // Each kind should open a bag about 1/7 of the time.
    let pieces = deal(99, 7 * 7000);
    let mut firsts = [0u32; 7];
    for chunk in pieces.chunks(7) {
        firsts[PieceKind::ALL.iter().position(|&k| k == chunk[0]).unwrap()] += 1;
    }
    for count in firsts {
        assert!((850..1150).contains(&count), "{firsts:?}");
    }
}
