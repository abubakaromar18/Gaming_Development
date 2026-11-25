using Platformer.Core.Input;
using Platformer.Core.Physics;
using Platformer.Core.World;

namespace Platformer.Core.Controller;

/// <summary>Coarse state of the jump arc, for animation and the debug HUD.</summary>
public enum MovementPhase
{
    Grounded,
    Rising,
    Apex,
    Falling,
}

/// <summary>
/// A kinematic platformer character: velocity is computed from input and tuning by hand
/// rather than by a physics engine, then the body is moved through the tile map one axis
/// at a time. Kinematic control is what lets the feel tricks below exist at all; a
/// rigid-body simulation would fight every one of them.
///
/// Each <see cref="Step"/> runs, in order:
///  1. age the coyote and buffer timers, and record a new jump press in the buffer;
///  2. jump if a press is live (this step or buffered) and the character is grounded or
///     inside its coyote window;
///  3. accelerate horizontally toward the input's target speed;
///  4. cut upward speed if the jump was released early (variable height), then apply
///     gravity, scaled down at the apex of a held jump and up when falling;
///  5. move X then Y through the map, with corner correction on head bumps, and update
///     grounded state.
/// </summary>
public sealed class CharacterController
{
    /// <summary>Axis values smaller than this count as no input, to absorb stick drift.</summary>
    public const float AxisDeadzone = 0.01f;

    private readonly TileMap _map;
    private readonly List<ControllerEvent> _events = new();
    private ControllerTuning _tuning;
    private Aabb _body;
    private float _coyoteRemaining;
    private float _bufferRemaining;
    private bool _jumpCut;

    public CharacterController(ControllerTuning tuning, TileMap map, Vec2 feet)
    {
        ArgumentNullException.ThrowIfNull(tuning);
        _map = map ?? throw new ArgumentNullException(nameof(map));
        _tuning = tuning.Validated();
        Teleport(feet);
    }

    /// <summary>Live-swappable tuning; validated on assignment.</summary>
    public ControllerTuning Tuning
    {
        get => _tuning;
        set
        {
            ArgumentNullException.ThrowIfNull(value);
            if (value.BodyWidth != _tuning.BodyWidth || value.BodyHeight != _tuning.BodyHeight)
            {
                throw new ArgumentException("Body size can't change at runtime; create a new controller instead.");
            }
            _tuning = value.Validated();
        }
    }

    public Aabb Body => _body;
    public Vec2 Velocity { get; private set; }
    public bool Grounded { get; private set; }

    /// <summary>True from take-off until landing, if the character left the ground by jumping.</summary>
    public bool IsJumping { get; private set; }

    /// <summary>True on steps where apex hang applied (reduced gravity, speed bonus).</summary>
    public bool InApex { get; private set; }

    /// <summary>+1 facing right, -1 facing left; follows the last non-zero input.</summary>
    public int Facing { get; private set; } = 1;

    public float CoyoteRemaining => _coyoteRemaining;
    public float BufferRemaining => _bufferRemaining;

    /// <summary>Bottom-centre of the body: the point that stands on the ground.</summary>
    public Vec2 Feet => new(_body.X + _body.Width / 2f, _body.Bottom);

    public MovementPhase Phase =>
        Grounded ? MovementPhase.Grounded
        : InApex ? MovementPhase.Apex
        : Velocity.Y < 0f ? MovementPhase.Rising
        : MovementPhase.Falling;

    /// <summary>What happened during the most recent <see cref="Step"/>.</summary>
    public IReadOnlyList<ControllerEvent> Events => _events;

    /// <summary>Place the body with its feet at <paramref name="feet"/> and reset all motion state.</summary>
    public void Teleport(Vec2 feet)
    {
        _body = new Aabb(feet.X - _tuning.BodyWidth / 2f, feet.Y - _tuning.BodyHeight, _tuning.BodyWidth, _tuning.BodyHeight);
        if (_map.Overlaps(_body))
        {
            throw new ArgumentException($"Body at feet {feet} would be inside a solid tile.", nameof(feet));
        }
        Velocity = Vec2.Zero;
        Grounded = TileCollider.MoveY(_map, _body, 1f).Hit;
        IsJumping = false;
        InApex = false;
        _jumpCut = false;
        _coyoteRemaining = 0f;
        _bufferRemaining = 0f;
        _events.Clear();
    }

    public void Step(ControllerInput input, float dt)
    {
        if (!(dt > 0f) || float.IsInfinity(dt))
        {
            throw new ArgumentOutOfRangeException(nameof(dt), dt, "Step needs a positive, finite dt.");
        }
        _events.Clear();
        input = input.Sanitized();
        var t = _tuning;

        // 1. Timers. Both count down every step; a fresh press refills the buffer.
        _coyoteRemaining = MathF.Max(0f, _coyoteRemaining - dt);
        _bufferRemaining = MathF.Max(0f, _bufferRemaining - dt);
        if (input.JumpPressed)
        {
            _bufferRemaining = t.JumpBufferTime;
        }

        // 2. Jump. A press is live this step even with a zero-length buffer, so turning
        //    buffering off never eats a press made while standing.
        var vx = Velocity.X;
        var vy = Velocity.Y;
        var pressLive = input.JumpPressed || _bufferRemaining > 0f;
        var canJump = Grounded || _coyoteRemaining > 0f;
        var jumpedThisStep = false;
        if (pressLive && canJump)
        {
            var assist = JumpAssist.None;
            if (!Grounded)
            {
                assist |= JumpAssist.Coyote;
            }
            if (!input.JumpPressed)
            {
                assist |= JumpAssist.Buffered;
            }
            vy = -t.JumpVelocity;
            Grounded = false;
            IsJumping = true;
            _jumpCut = false;
            _coyoteRemaining = 0f;
            _bufferRemaining = 0f;
            jumpedThisStep = true;
            _events.Add(new ControllerEvent(ControllerEventKind.Jumped, Feet, t.JumpVelocity, assist));
        }

        // Apex hang only rewards a held jump near the top of its arc.
        InApex = IsJumping && input.JumpHeld && MathF.Abs(vy) < t.ApexThreshold;

        // 3. Horizontal: approach the target speed at a rate chosen by context.
        var hasInput = MathF.Abs(input.MoveX) > AxisDeadzone;
        if (hasInput)
        {
            Facing = input.MoveX > 0f ? 1 : -1;
        }
        var target = input.MoveX * t.MaxRunSpeed * (InApex ? t.ApexSpeedBonus : 1f);
        float accel;
        if (Grounded)
        {
            var turning = hasInput && vx != 0f && MathF.Sign(target) != MathF.Sign(vx);
            accel = !hasInput ? t.GroundDeceleration : turning ? t.TurnAcceleration : t.GroundAcceleration;
        }
        else
        {
            accel = hasInput ? t.AirAcceleration : t.AirDeceleration;
        }
        vx = MoveTowards(vx, target, accel * dt);

        // 4. Vertical. Releasing jump while rising cuts the remaining rise once.
        if (IsJumping && vy < 0f && !input.JumpHeld && !_jumpCut)
        {
            _jumpCut = true;
            if (t.JumpCutMultiplier < 1f)
            {
                vy *= t.JumpCutMultiplier;
                _events.Add(new ControllerEvent(ControllerEventKind.JumpCut, Feet, vy));
            }
        }
        // The multipliers compose, so the hang straddles the top of the arc on both sides,
        // and an apex multiplier of 1 leaves the arc exactly as if hang didn't exist.
        var gravityScale = (vy > 0f ? t.FallGravityMultiplier : 1f) * (InApex ? t.ApexGravityMultiplier : 1f);
        vy = MathF.Min(vy + t.Gravity * gravityScale * dt, t.MaxFallSpeed);

        // 5. Move: X first, then Y (semi-implicit Euler: new velocity, then position).
        var wasGrounded = Grounded;
        var moveX = TileCollider.MoveX(_map, _body, vx * dt);
        _body = moveX.Box;
        if (moveX.Hit)
        {
            vx = 0f;
        }

        var dy = vy * dt;
        var moveY = TileCollider.MoveY(_map, _body, dy);
        if (moveY.Hit && dy < 0f && t.CornerCorrection > 0)
        {
            var nudge = TileCollider.FindCeilingNudge(_map, _body, dy, t.CornerCorrection);
            if (nudge is { } offset)
            {
                _body = _body.Offset(offset, 0f);
                moveY = TileCollider.MoveY(_map, _body, dy);
                _events.Add(new ControllerEvent(ControllerEventKind.CornerCorrected, Feet, offset));
            }
        }
        _body = moveY.Box;

        if (moveY.Hit && dy > 0f)
        {
            if (!wasGrounded)
            {
                _events.Add(new ControllerEvent(ControllerEventKind.Landed, Feet, vy));
            }
            Grounded = true;
            IsJumping = false;
            InApex = false;
            vy = 0f;
        }
        else if (moveY.Hit)
        {
            vy = 0f;
            _events.Add(new ControllerEvent(ControllerEventKind.HeadBump, Feet));
        }
        else
        {
            Grounded = false;
            if (wasGrounded && !jumpedThisStep)
            {
                // Walked off a ledge: open the coyote window.
                _coyoteRemaining = t.CoyoteTime;
            }
        }

        Velocity = new Vec2(vx, vy);
    }

    private static float MoveTowards(float current, float target, float maxDelta) =>
        MathF.Abs(target - current) <= maxDelta ? target : current + MathF.Sign(target - current) * maxDelta;
}
