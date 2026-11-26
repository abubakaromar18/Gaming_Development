//! A minimal 2D vector, so the core needs no math crate.

use std::ops::{Add, AddAssign, Mul, Neg, Sub, SubAssign};

#[derive(Debug, Clone, Copy, PartialEq, Default)]
pub struct Vec2 {
    pub x: f32,
    pub y: f32,
}

impl Vec2 {
    pub const ZERO: Vec2 = Vec2 { x: 0.0, y: 0.0 };

    pub const fn new(x: f32, y: f32) -> Vec2 {
        Vec2 { x, y }
    }

    /// Unit vector pointing at `radians` (0 = +x, counter-clockwise in a
    /// y-up frame; clockwise on a y-down screen).
    pub fn from_angle(radians: f32) -> Vec2 {
        Vec2::new(radians.cos(), radians.sin())
    }

    pub fn dot(self, other: Vec2) -> f32 {
        self.x * other.x + self.y * other.y
    }

    /// z-component of the 3D cross product; its sign says which side
    /// `other` lies on.
    pub fn cross(self, other: Vec2) -> f32 {
        self.x * other.y - self.y * other.x
    }

    pub fn length_squared(self) -> f32 {
        self.dot(self)
    }

    pub fn length(self) -> f32 {
        self.length_squared().sqrt()
    }

    pub fn distance(self, other: Vec2) -> f32 {
        (self - other).length()
    }

    /// Unit vector in the same direction, or zero for a (near) zero vector.
    pub fn normalized_or_zero(self) -> Vec2 {
        let len = self.length();
        if len > 1e-6 {
            self * (1.0 / len)
        } else {
            Vec2::ZERO
        }
    }

    /// Shortens the vector to `max` if it is longer; direction is kept.
    pub fn clamp_length(self, max: f32) -> Vec2 {
        let len = self.length();
        if len > max && len > 0.0 {
            self * (max / len)
        } else {
            self
        }
    }

    pub fn angle(self) -> f32 {
        self.y.atan2(self.x)
    }

    pub fn rotate(self, radians: f32) -> Vec2 {
        let (s, c) = radians.sin_cos();
        Vec2::new(self.x * c - self.y * s, self.x * s + self.y * c)
    }

    /// Signed angle that rotates `self` onto `other`, in `[-PI, PI]`.
    pub fn angle_to(self, other: Vec2) -> f32 {
        self.cross(other).atan2(self.dot(other))
    }

    /// Moves toward `target` by at most `max_delta`, never overshooting.
    pub fn move_towards(self, target: Vec2, max_delta: f32) -> Vec2 {
        let delta = target - self;
        let dist = delta.length();
        if dist <= max_delta || dist == 0.0 {
            target
        } else {
            self + delta * (max_delta / dist)
        }
    }
}

impl Add for Vec2 {
    type Output = Vec2;
    fn add(self, o: Vec2) -> Vec2 {
        Vec2::new(self.x + o.x, self.y + o.y)
    }
}

impl AddAssign for Vec2 {
    fn add_assign(&mut self, o: Vec2) {
        *self = *self + o;
    }
}

impl Sub for Vec2 {
    type Output = Vec2;
    fn sub(self, o: Vec2) -> Vec2 {
        Vec2::new(self.x - o.x, self.y - o.y)
    }
}

impl SubAssign for Vec2 {
    fn sub_assign(&mut self, o: Vec2) {
        *self = *self - o;
    }
}

impl Mul<f32> for Vec2 {
    type Output = Vec2;
    fn mul(self, s: f32) -> Vec2 {
        Vec2::new(self.x * s, self.y * s)
    }
}

impl Neg for Vec2 {
    type Output = Vec2;
    fn neg(self) -> Vec2 {
        Vec2::new(-self.x, -self.y)
    }
}

/// Moves a scalar toward `target` by at most `max_delta`.
pub fn approach(current: f32, target: f32, max_delta: f32) -> f32 {
    if current < target {
        (current + max_delta).min(target)
    } else {
        (current - max_delta).max(target)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::f32::consts::FRAC_PI_2;

    #[test]
    fn angle_to_is_signed() {
        let right = Vec2::new(1.0, 0.0);
        let up = Vec2::new(0.0, 1.0);
        assert!((right.angle_to(up) - FRAC_PI_2).abs() < 1e-6);
        assert!((up.angle_to(right) + FRAC_PI_2).abs() < 1e-6);
    }

    #[test]
    fn move_towards_never_overshoots() {
        let a = Vec2::new(0.0, 0.0);
        let b = Vec2::new(3.0, 4.0);
        assert_eq!(a.move_towards(b, 10.0), b);
        assert!((a.move_towards(b, 2.5).length() - 2.5).abs() < 1e-6);
    }

    #[test]
    fn normalizing_zero_is_safe() {
        assert_eq!(Vec2::ZERO.normalized_or_zero(), Vec2::ZERO);
    }
}
