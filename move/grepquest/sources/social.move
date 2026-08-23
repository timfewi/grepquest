// SPDX-License-Identifier: MIT

module grepquest::social;

use sui::event;
use sui::object::{Self, UID};
use sui::table::{Self, Table};
use sui::transfer;
use sui::tx_context::{Self, TxContext};

const E_INVALID_DIGEST: u64 = 1;
const E_INVALID_EPOCH_RANGE: u64 = 2;
const E_NOT_OWNER: u64 = 3;
const E_SEASON_CLOSED: u64 = 4;
const E_INVALID_PROOF_TIER: u64 = 5;
const E_INVALID_METRIC: u64 = 6;
const E_SEASON_MISMATCH: u64 = 7;

/// Package authority. Season creation is deliberately separate from player
/// score writes; players never contend on one mutable leaderboard object.
public struct AdminCap has key {
    id: UID,
}

/// Least-privilege capability for CI/official verifiers. Possession authorizes
/// a verified event but does not grant season administration.
public struct VerifierCap has key {
    id: UID,
}

/// Immutable rules and content identity for one season.
public struct Season has key {
    id: UID,
    season_id: vector<u8>,
    pack_sha256: vector<u8>,
    rules_sha256: vector<u8>,
    start_epoch: u64,
    end_epoch: u64,
}

public struct SeasonBest has store {
    score: u64,
    receipt_sha256: vector<u8>,
    proof_tier: u8,
}

/// Address-owned profile. Its Table stores at most one best record per season.
public struct PlayerProfile has key {
    id: UID,
    owner: address,
    mastery_bitmap: vector<u8>,
    level: u64,
    best_season_score: u64,
    latest_receipt_sha256: vector<u8>,
    proof_tier: u8,
    season_bests: Table<vector<u8>, SeasonBest>,
}

public struct Achievement has key {
    id: UID,
    owner: address,
    achievement_type: vector<u8>,
    season_id: vector<u8>,
    receipt_sha256: vector<u8>,
}

public struct LoadoutSnapshot has key {
    id: UID,
    owner: address,
    item_ids_or_hashes: vector<vector<u8>>,
    stats_hash: vector<u8>,
    receipt_sha256: vector<u8>,
}

/// Public board material. Consumers must keep proof tiers separate and dedupe
/// each player locally; an event does not turn a self-attestation into truth.
public struct ScoreSubmitted has copy, drop {
    player: address,
    season_id: vector<u8>,
    score: u64,
    level: u64,
    receipt_sha256: vector<u8>,
    proof_tier: u8,
    accuracy: u64,
    recall: u64,
    no_hint_clears: u64,
    elapsed_ms: u64,
}

fun init(ctx: &mut TxContext) {
    transfer::transfer(AdminCap { id: object::new(ctx) }, ctx.sender());
}

public fun create_season(
    _cap: &AdminCap,
    season_id: vector<u8>,
    pack_sha256: vector<u8>,
    rules_sha256: vector<u8>,
    start_epoch: u64,
    end_epoch: u64,
    ctx: &mut TxContext,
) {
    assert!(season_id.length() > 0 && season_id.length() <= 128, E_INVALID_DIGEST);
    assert!(pack_sha256.length() == 32 && rules_sha256.length() == 32, E_INVALID_DIGEST);
    assert!(start_epoch <= end_epoch, E_INVALID_EPOCH_RANGE);
    transfer::freeze_object(Season {
        id: object::new(ctx),
        season_id,
        pack_sha256,
        rules_sha256,
        start_epoch,
        end_epoch,
    });
}

public fun create_verifier_cap(_cap: &AdminCap, recipient: address, ctx: &mut TxContext) {
    transfer::transfer(VerifierCap { id: object::new(ctx) }, recipient);
}

public fun create_profile(ctx: &mut TxContext) {
    let owner = ctx.sender();
    transfer::transfer(PlayerProfile {
        id: object::new(ctx),
        owner,
        mastery_bitmap: vector[],
        level: 1,
        best_season_score: 0,
        latest_receipt_sha256: vector[],
        proof_tier: 0,
        season_bests: table::new(ctx),
    }, owner);
}

public fun submit_score(
    profile: &mut PlayerProfile,
    season: &Season,
    receipt_season_id: vector<u8>,
    receipt_pack_sha256: vector<u8>,
    receipt_rules_sha256: vector<u8>,
    mastery_bitmap: vector<u8>,
    level: u64,
    score: u64,
    receipt_sha256: vector<u8>,
    proof_tier: u8,
    accuracy: u64,
    recall: u64,
    no_hint_clears: u64,
    elapsed_ms: u64,
    ctx: &mut TxContext,
) {
    assert!(profile.owner == ctx.sender(), E_NOT_OWNER);
    assert!(ctx.epoch() >= season.start_epoch && ctx.epoch() <= season.end_epoch, E_SEASON_CLOSED);
    assert!(receipt_sha256.length() == 32 && receipt_pack_sha256.length() == 32 && receipt_rules_sha256.length() == 32, E_INVALID_DIGEST);
    assert!(receipt_season_id == season.season_id && receipt_pack_sha256 == season.pack_sha256 && receipt_rules_sha256 == season.rules_sha256, E_SEASON_MISMATCH);
    // A player wallet can only self-attest. Verified/official events require a
    // separate VerifierCap entry point below.
    assert!(proof_tier == 0, E_INVALID_PROOF_TIER);
    assert!(accuracy <= 100 && recall <= 100, E_INVALID_METRIC);

    let season_id = season.season_id;
    let should_update = if (!profile.season_bests.contains(copy season_id)) {
        true
    } else {
        let best = profile.season_bests.borrow(copy season_id);
        score > best.score || (score == best.score && proof_tier > best.proof_tier)
    };
    if (should_update) {
        if (profile.season_bests.contains(copy season_id)) {
            let best = profile.season_bests.borrow_mut(copy season_id);
            best.score = score;
            best.receipt_sha256 = copy receipt_sha256;
            best.proof_tier = proof_tier;
        } else {
            profile.season_bests.add(copy season_id, SeasonBest {
                score,
                receipt_sha256: copy receipt_sha256,
                proof_tier,
            });
        };
        profile.mastery_bitmap = mastery_bitmap;
        profile.level = level;
        profile.best_season_score = score;
        profile.latest_receipt_sha256 = copy receipt_sha256;
        profile.proof_tier = proof_tier;
    };

    event::emit(ScoreSubmitted {
        player: profile.owner,
        season_id,
        score,
        level,
        receipt_sha256,
        proof_tier,
        accuracy,
        recall,
        no_hint_clears,
        elapsed_ms,
    });
}

public fun submit_verified_score(
    _cap: &VerifierCap,
    player: address,
    season: &Season,
    receipt_season_id: vector<u8>,
    receipt_pack_sha256: vector<u8>,
    receipt_rules_sha256: vector<u8>,
    level: u64,
    score: u64,
    receipt_sha256: vector<u8>,
    proof_tier: u8,
    accuracy: u64,
    recall: u64,
    no_hint_clears: u64,
    elapsed_ms: u64,
    ctx: &TxContext,
) {
    assert!(ctx.epoch() >= season.start_epoch && ctx.epoch() <= season.end_epoch, E_SEASON_CLOSED);
    assert!(receipt_sha256.length() == 32 && receipt_pack_sha256.length() == 32 && receipt_rules_sha256.length() == 32, E_INVALID_DIGEST);
    assert!(receipt_season_id == season.season_id && receipt_pack_sha256 == season.pack_sha256 && receipt_rules_sha256 == season.rules_sha256, E_SEASON_MISMATCH);
    assert!(proof_tier == 1 || proof_tier == 2, E_INVALID_PROOF_TIER);
    assert!(accuracy <= 100 && recall <= 100, E_INVALID_METRIC);
    event::emit(ScoreSubmitted {
        player,
        season_id: copy season.season_id,
        score,
        level,
        receipt_sha256,
        proof_tier,
        accuracy,
        recall,
        no_hint_clears,
        elapsed_ms,
    });
}

public fun mint_achievement(
    profile: &PlayerProfile,
    season: &Season,
    achievement_type: vector<u8>,
    receipt_sha256: vector<u8>,
    ctx: &mut TxContext,
) {
    assert!(profile.owner == ctx.sender(), E_NOT_OWNER);
    assert!(receipt_sha256.length() == 32, E_INVALID_DIGEST);
    let owner = profile.owner;
    transfer::transfer(Achievement {
        id: object::new(ctx),
        owner,
        achievement_type,
        season_id: copy season.season_id,
        receipt_sha256,
    }, owner);
}

public fun snapshot_loadout(
    profile: &PlayerProfile,
    item_ids_or_hashes: vector<vector<u8>>,
    stats_hash: vector<u8>,
    receipt_sha256: vector<u8>,
    ctx: &mut TxContext,
) {
    assert!(profile.owner == ctx.sender(), E_NOT_OWNER);
    assert!(stats_hash.length() == 32 && receipt_sha256.length() == 32, E_INVALID_DIGEST);
    let owner = profile.owner;
    transfer::transfer(LoadoutSnapshot {
        id: object::new(ctx),
        owner,
        item_ids_or_hashes,
        stats_hash,
        receipt_sha256,
    }, owner);
}

public fun profile_owner(profile: &PlayerProfile): address { profile.owner }
public fun best_season_score(profile: &PlayerProfile): u64 { profile.best_season_score }
public fun latest_receipt_sha256(profile: &PlayerProfile): vector<u8> { profile.latest_receipt_sha256 }

#[test_only]
public fun init_for_testing(ctx: &mut TxContext) { init(ctx) }
